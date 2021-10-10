package sshd

import (
	"strconv"

	"github.com/ibuler/ssh"

	"cocogo/pkg/auth"
	"cocogo/pkg/config"
	"cocogo/pkg/handler"
	"cocogo/pkg/logger"
)

const version = "coco-v1.4"

var (
	conf = config.Conf
)

func StartServer() {
	logger.Debug("Load host access key")
	hostKey := HostKey{Value: conf.HostKey, Path: conf.HostKeyFile}
	signer, err := hostKey.Load()
	if err != nil {
		logger.Fatal("Load access key error: %s", err)
	}

	srv := ssh.Server{
		Addr:                       conf.BindHost + ":" + strconv.Itoa(conf.SSHPort),
		PasswordHandler:            auth.CheckUserPassword,
		PublicKeyHandler:           auth.CheckUserPublicKey,
		KeyboardInteractiveHandler: auth.CheckMFA,
		HostSigners:                []ssh.Signer{signer},
		Version:                    version,
		Handler:                    handler.SessionHandler,
		SubsystemHandlers:          map[string]ssh.SubsystemHandler{},
	}
	srv.SetSubsystemHandler("sftp", handler.SftpHandler)
	logger.Fatal(srv.ListenAndServe())
}
