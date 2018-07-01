
package main

import (
	"github.com/HotelsDotCom/flyte-client/client"
	"github.com/HotelsDotCom/flyte-client/flyte"
	"net/url"
	"os"
	"time"
	"github.com/HotelsDotCom/go-logger"
	"github.com/jollinshead/flyte-pack/config"
)

func main() {
	flyteHost, exists := os.LookupEnv("FLYTE_API")
	if !exists {
		logger.Fatal("FLYTE_API environment variable is not set")
	}

	configPath, exists := os.LookupEnv("FLYTE_PACK_CONFIG")
	if !exists {
		logger.Fatal("FLYTE_PACK_CONFIG environment variable is not set")
	}

	packDef, err := config.NewPackDef(configPath)
	if err != nil {
		logger.Fatalf("could not generate pack err: %s", err)
	}

	hostUrl := getUrl(flyteHost)
	p := flyte.NewPack(packDef, client.NewClient(hostUrl, 10*time.Second))
	p.Start()
	// Waits indefinitely
	select {}
}

func getUrl(rawUrl string) *url.URL {
	url, err := url.Parse(rawUrl)
	if err != nil {
		logger.Fatalf("%s is not a valid url", rawUrl)
	}
	return url
}



