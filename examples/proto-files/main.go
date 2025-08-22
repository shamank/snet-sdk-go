package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	c := config.Config{
		RPCAddr: "",
		// You can unfill the private key if you do not want to call the services
		PrivateKey: "",
	}

	// creating a new SDK core
	snetSDK := sdk.NewSDK(&c)

	// create a service client
	service, err := snetSDK.NewServiceClient("", "", "default_group")
	if err != nil {
		log.Fatalln(err)
	}

	// just get proto files as a map
	protoFiles := service.ProtoFiles()
	//fmt.Println(protoFiles)
	fmt.Println(protoFiles["main.proto"])

	err = service.SaveProtoFiles("./files/")
	if err != nil {
		log.Fatalln(err)
	}

	err = service.SaveProtoFilesZip("protos.zip")
	if err != nil {
		log.Fatalln(err)
	}

	snetSDK.Close()
}
