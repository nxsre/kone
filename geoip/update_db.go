package geoip

import (
	"github.com/maxmind/geoipupdate/pkg/geoipupdate"
	"github.com/maxmind/geoipupdate/pkg/geoipupdate/database"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// 注册账号（不能翻墙，不能用VPN）：https://www.maxmind.com/en/geolite2/signup
// 登录：https://www.maxmind.com/en/account/login
// 生成License Key：Services  --> My License Key
// jBNphFHzkzJ9A9Nn

func init() {
	//UpdateDB()
}

func UpdateDB() {
	os.Mkdir("mmdbs", 0777)
	proxy, _ := url.Parse("socks5://172.18.5.7:7891")
	config := &geoipupdate.Config{
		AccountID:         224237,
		DatabaseDirectory: "mmdbs",
		LicenseKey:        "jBNphFHzkzJ9A9Nn",
		LockFile:          "/tmp/up.lock",
		URL:               "https://" + "updates.maxmind.com",
		EditionIDs:        []string{"GeoLite2-Country"},
		Proxy:             proxy,
		PreserveFileTimes: false,
		Verbose:           true,
	}
	client := geoipupdate.NewClient(config)

	log.Println(run(client, config))

}

func run(client *http.Client, config *geoipupdate.Config) error {
	dbReader := database.NewHTTPDatabaseReader(client, config)

	for _, editionID := range config.EditionIDs {
		filename, err := geoipupdate.GetFilename(config, editionID, client)
		if err != nil {
			return errors.Wrap(err, "error retrieving filename")
		}
		filePath := filepath.Join(config.DatabaseDirectory, filename)
		dbWriter, err := database.NewLocalFileDatabaseWriter(filePath, config.LockFile, config.Verbose)
		if err != nil {
			return errors.Wrap(err, "error creating database writer")
		}
		if err := dbReader.Get(dbWriter, editionID); err != nil {
			return err
		}
	}
	return nil
}
