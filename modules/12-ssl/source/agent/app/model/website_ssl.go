// =============================================================================
// 模块: SSL 证书管理 (agent/app/model/website_ssl.go)
// 文件: website_ssl.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

import (
	"fmt"
	"path"
	"time"

	"github.com/1Panel-dev/1Panel/agent/global"
)

// WebsiteSSL (struct)
type WebsiteSSL struct {
	BaseModel
	PrimaryDomain  string    `json:"primaryDomain"`
	PrivateKey     string    `json:"privateKey"`
	Pem            string    `json:"pem"`
	Domains        string    `json:"domains"`
	CertURL        string    `json:"certURL"`
	Type           string    `json:"type"`
	Provider       string    `json:"provider"`
	Organization   string    `json:"organization"`
	DnsAccountID   uint      `json:"dnsAccountId"`
	AcmeAccountID  uint      `gorm:"column:acme_account_id" json:"acmeAccountId" `
	CaID           uint      `json:"caId"`
	AutoRenew      bool      `json:"autoRenew"`
	ExpireDate     time.Time `json:"expireDate"`
	StartDate      time.Time `json:"startDate"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	KeyType        string    `json:"keyType"`
	PushDir        bool      `json:"pushDir"`
	Dir            string    `json:"dir"`
	Description    string    `json:"description"`
	SkipDNS        bool      `json:"skipDNS"`
	Nameserver1    string    `json:"nameserver1"`
	Nameserver2    string    `json:"nameserver2"`
	DisableCNAME   bool      `json:"disableCNAME"`
	ExecShell      bool      `json:"execShell"`
	Shell          string    `json:"shell"`
	MasterSSLID    uint      `json:"masterSslId"`
	Nodes          string    `json:"nodes"`
	PushNode       bool      `json:"pushNode"`
	PrivateKeyPath string    `json:"privateKeyPath"`
	CertPath       string    `json:"certPath"`
	IsIp           bool      `json:"isIP"`

	AcmeAccount WebsiteAcmeAccount `json:"acmeAccount" gorm:"-:migration"`
	DnsAccount  WebsiteDnsAccount  `json:"dnsAccount" gorm:"-:migration"`
	Websites    []Website          `json:"websites" gorm:"-:migration"`
}

func (w WebsiteSSL) TableName() string {
	return "website_ssls"
}

func (w WebsiteSSL) GetLogPath() string {
	return path.Join(global.Dir.SSLLogDir, fmt.Sprintf("%s-ssl-%d.log", w.PrimaryDomain, w.ID))
}
