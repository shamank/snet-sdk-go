## How to read proto files of service

You can inspect a service API by fetching its proto files.


```go

    // ...

	service, err := snetSDK.NewServiceClient("orgID", "serviceID", "default_group")
	if err != nil {
		log.Fatalln(err)
	}

	// just get proto files as a map
	protoFiles := service.ProtoFiles()
	fmt.Println(protoFiles)
	fmt.Println(protoFiles["main.proto"])

	// save protos in dir
	err = service.ProtoFiles().Save("./files/")
	if err != nil {
		log.Fatalln(err)
	}

	// save protos as zip archive
	err = service.ProtoFiles().SaveAsZip("protos.zip")
	if err != nil {
		log.Fatalln(err)
	}
}
```