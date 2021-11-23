package handler

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/gliderlabs/ssh"

	"cocogo/pkg/config"
	"cocogo/pkg/i18n"
	"cocogo/pkg/logger"
	"cocogo/pkg/utils"
)

var defaultTitle string
var menu Menu

type MenuItem struct {
	id       int
	instruct string
	helpText string
	showText string
}

func (mi *MenuItem) Text() string {
	if mi.showText != "" {
		return mi.showText
	}
	cm := ColorMeta{GreenBoldColor: "\033[1;32m", ColorEnd: "\033[0m"}
	line := fmt.Sprintf(i18n.T("\t%d) Enter {{.GreenBoldColor}}%s{{.ColorEnd}} to %s.%s"), mi.id, mi.instruct, mi.helpText, "\r\n")
	tmpl := template.Must(template.New("item").Parse(line))

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, cm)
	if err != nil {
		logger.Error(err)
	}
	mi.showText = string(buf.Bytes())
	return mi.showText
}

type Menu []MenuItem

func init() {
	defaultTitle = utils.WrapperTitle(i18n.T("Welcome to use Jumpserver open source fortress system"))
	menu = Menu{
		{id: 1, instruct: "ID", helpText: i18n.T("directly login")},
		{id: 2, instruct: i18n.T("part IP, Hostname, Comment"), helpText: i18n.T("to search login if unique")},
		{id: 3, instruct: i18n.T("/ + IP, Hostname, Comment"), helpText: i18n.T("to search, such as: /192.168")},
		{id: 4, instruct: "p", helpText: i18n.T("display the host you have permission")},
		{id: 5, instruct: "g", helpText: "display the node that you have permission"},
		{id: 6, instruct: "r", helpText: "refresh your assets and nodes"},
		{id: 7, instruct: "h", helpText: "print help"},
		{id: 8, instruct: "q", helpText: "exit"},
	}
}

type ColorMeta struct {
	GreenBoldColor string
	ColorEnd       string
}

func displayBanner(sess ssh.Session, user string) {
	title := defaultTitle
	if config.Conf.HeaderTitle != "" {
		title = config.Conf.HeaderTitle
	}

	prefix := utils.CharClear + utils.CharTab + utils.CharTab
	suffix := utils.CharNewLine + utils.CharNewLine
	welcomeMsg := prefix + utils.WrapperTitle(user+",") + "  " + title + suffix
	_, err := io.WriteString(sess, welcomeMsg)
	if err != nil {
		logger.Error("Send to client error, %s", err)
		return
	}
	for _, v := range menu {
		utils.IgnoreErrWriteString(sess, v.Text())
	}
}
