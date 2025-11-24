package main

import (
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	c := config.Config{
		RPCAddr:    "https://sepolia.infura.io/v3/",
		PrivateKey: "",
		Debug:      true,
	}

	// creating new SDK core
	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("", "", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	// just get proto files as a map
	//protoFiles := service.ProtoFiles()
	//fmt.Println(protoFiles)
	//fmt.Println(protoFiles["main.proto"])

	err = service.ProtoFiles().Save("./files/")
	if err != nil {
		log.Fatalln(err)
	}

	err = service.ProtoFiles().SaveAsZip("protos.zip")
	if err != nil {
		log.Fatalln(err)
	}

	snetSDK.Close()
}
