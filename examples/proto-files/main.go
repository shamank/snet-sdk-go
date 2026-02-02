package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	c := config.Config{
		RPCAddr:    "wss://sepolia.infura.io/ws/v3/your_infura_project_id",
		PrivateKey: "your_private_key",
		Debug:      true,
		Network:    config.Sepolia,
	}

	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("your_org_id", "your_service_id", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	// Get proto files as a map
	protoFiles := service.ProtoFiles()
	fmt.Println(protoFiles)

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
