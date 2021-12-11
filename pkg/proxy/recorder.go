package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cocogo/pkg/common"
	"cocogo/pkg/config"
	"cocogo/pkg/logger"
	"cocogo/pkg/model"
)

var conf = config.Conf

type CommandRecorder struct {
	Session *SwitchSession
	storage CommandStorage

	queue chan *model.Command
}

func NewCommandRecorder(sess *SwitchSession) (recorder *CommandRecorder) {
	storage := NewCommandStorage()
	recorder = &CommandRecorder{Session: sess, queue: make(chan *model.Command, 10), storage: storage}
	go recorder.record()
	recorder.Start()
	return recorder
}

func NewReplyRecord(sess *SwitchSession) (recorder *ReplyRecorder) {
	storage := NewReplayStorage()
	srvStorage := &ServerReplayStorage{}
	recorder = &ReplyRecorder{SessionID: sess.Id, storage: storage, backOffStorage: srvStorage}
	recorder.Start()
	return recorder
}

func (c *CommandRecorder) Record(command [2]string) {
	if command[0] == "" && command[1] == "" {
		return
	}
	cmd := &model.Command{
		SessionId:  c.Session.Id,
		OrgId:      c.Session.Org,
		Input:      command[0],
		Output:     command[1],
		User:       c.Session.User,
		Server:     c.Session.Server,
		SystemUser: c.Session.SystemUser,
		Timestamp:  time.Now().Unix(),
	}
	c.queue <- cmd
}

func (c *CommandRecorder) Start() {
}

func (c *CommandRecorder) End() {
	close(c.queue)
}

func (c *CommandRecorder) record() {
	cmdList := make([]*model.Command, 0)
	for {
		select {
		case p, ok := <-c.queue:
			if !ok {
				logger.Debug("Session command recorder close: ", c.Session.Id)
				return
			}
			cmdList = append(cmdList, p)
			if len(cmdList) < 5 {
				continue
			}
		case <-time.After(time.Second * 5):
			if len(cmdList) == 0 {
				continue
			}
		}

		err := c.storage.BulkSave(cmdList)
		if err == nil {
			cmdList = cmdList[:0]
			continue
		}
		if len(cmdList) > 10 {
			cmdList = cmdList[1:]
		}
	}
}

type ReplyRecorder struct {
	SessionID string

	absFilePath   string
	absGzFilePath string
	target        string
	file          *os.File
	timeStartNano int64

	storage        ReplayStorage
	backOffStorage ReplayStorage
}

func (r *ReplyRecorder) Record(b []byte) {
	if len(b) > 0 {
		delta := float64(time.Now().UnixNano()-r.timeStartNano) / 1000 / 1000 / 1000
		data, _ := json.Marshal(string(b))
		_, _ = r.file.WriteString(fmt.Sprintf(`"%.3f":%s`, delta, data))
	}
}

func (r *ReplyRecorder) Start() {
	rootPath := conf.RootPath
	today := time.Now().UTC().Format("2006-01-02")
	gzFileName := r.SessionID + ".replay.gz"
	replayDir := filepath.Join(rootPath, "data", "replays", today)

	r.absFilePath = filepath.Join(replayDir, r.SessionID)
	r.absGzFilePath = filepath.Join(replayDir, today, gzFileName)
	r.target = strings.Join([]string{today, gzFileName}, "/")
	r.timeStartNano = time.Now().UnixNano()

	err := common.EnsureDirExist(replayDir)
	if err != nil {
		logger.Errorf("Create dir %s error: %s\n", replayDir, err)
		return
	}

	logger.Debug("Replay file path: ", r.absFilePath)
	r.file, err = os.Create(r.absFilePath)
	if err != nil {
		logger.Errorf("Create file %s error: %s\n", r.absFilePath, err)
	}
	_, _ = r.file.Write([]byte("{"))
}

func (r *ReplyRecorder) End() {
	_ = r.file.Close()
	go r.uploadReplay()
}

func (r *ReplyRecorder) uploadReplay() {
	maxRetry := 3
	if !common.FileExists(r.absFilePath) {
		logger.Debug("Replay file not found, passed: ", r.absFilePath)
		return
	}
	if stat, err := os.Stat(r.absFilePath); err == nil && stat.Size() == 0 {
		logger.Debug("Replay file is empty, removed: ", r.absFilePath)
		_ = os.Remove(r.absFilePath)
		return
	}
	if !common.FileExists(r.absGzFilePath) {
		logger.Debug("Compress replay file: ", r.absFilePath)
		_ = common.GzipCompressFile(r.absFilePath, r.absGzFilePath)
		_ = os.Remove(r.absFilePath)
	}

	for i := 0; i <= maxRetry; i++ {
		logger.Debug("Upload replay file: ", r.absGzFilePath)
		err := r.storage.Upload(r.absGzFilePath, r.target)
		if err == nil {
			_ = os.Remove(r.absGzFilePath)
			break
		}
		// 如果还是失败，使用备用storage再传一次
		if i == maxRetry {
			logger.Errorf("Using back off storage retry upload")
			r.storage = r.backOffStorage
			r.uploadReplay()
			break
		}
	}
}
