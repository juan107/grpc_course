package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"hello/hellopb"
	"io"
	"io/ioutil"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	fmt.Println("Go Clien Is Running")

	caCert, err := ioutil.ReadFile("../ssl/server.crt")
	if err != nil {
		log.Fatalln(err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(caCert)

	tlsConf := &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		ServerName:         "localhost",
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)),
	}

	conn, err := grpc.Dial(":50051", opts...)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	c := hellopb.NewHelloServiceClient(conn)
	helloUnary(c)
	//helloServerStreaming(c)
	//goodByeClientStreaming(c)
	//goodbyeBidiStreaming(c)

}

// Unary Pattern
func helloUnary(c hellopb.HelloServiceClient) {
	fmt.Println("Starting Unary RPC Hello")

	req := &hellopb.HelloRequest{
		Hello: &hellopb.Hello{
			FirstName: "Juan",
			Prefix:    "Señor",
		},
	}
	res, err := c.Hello(context.Background(), req)
	if err != nil {
		log.Fatalf("Error calling Server RPC %v", err)
	}
	log.Printf("Response Hello: %v", res.CustomHello)
}

// Server Streaming
func helloServerStreaming(c hellopb.HelloServiceClient) {
	fmt.Println("Starting Calling Server Streaming RPC Hello Many Languages")

	req := &hellopb.HelloManyLanguagesRequest{
		Hello: &hellopb.Hello{
			FirstName: "Juan",
			Prefix:    "Señor",
		},
	}
	res, err := c.HelloManyLanguages(context.Background(), req)
	if err != nil {
		log.Fatalf("Error calling Server RPC %v", err)
	}

	for {
		mess, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error getting the response Server RPC %v", err)
		}
		log.Printf("Response Server Streaming: %v", mess.GetHelloLanguage())
	}
}

// Client Server Streaming
func goodByeClientStreaming(c hellopb.HelloServiceClient) {
	fmt.Println("Starting Calling Client Streaming RPC Goodbye Function")
	reqs := []*hellopb.HellosGoodbyeRequest{
		{
			Hello: &hellopb.Hello{
				FirstName: "Campoelias",
				Prefix:    "Doctor",
			},
		},
		{
			Hello: &hellopb.Hello{
				FirstName: "Ulises",
				Prefix:    "Doctor",
			},
		},
		{
			Hello: &hellopb.Hello{
				FirstName: "Eutimio",
				Prefix:    "Doctor",
			},
		},
	}

	stream, err := c.HellosGoodbye(context.Background())

	if err != nil {
		log.Fatalf("Error Calling Hellos Goodbye Function %v", err)
	}
	for _, req := range reqs {
		fmt.Println("Sending Request: ", req)
		stream.Send(req)
		time.Sleep(1000 * time.Millisecond)
	}
	goodbye, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Error Recieving Hellos Goodbye Function %v", err)
	}
	fmt.Println(goodbye)
}

func goodbyeBidiStreaming(c hellopb.HelloServiceClient) {
	fmt.Println("Starting Calling Bidirectional Streaming Function")

	//crear el stream para llamar server
	str, err := c.Goodbye(context.Background())
	reqs := []*hellopb.GoodbyeRequest{
		{
			Hello: &hellopb.Hello{
				FirstName: "Mark",
				Prefix:    "Joven",
			},
		},
		{
			Hello: &hellopb.Hello{
				FirstName: "Jhon",
				Prefix:    "Dr",
			},
		},
		{
			Hello: &hellopb.Hello{
				FirstName: "Karla",
				Prefix:    "Ms",
			},
		},
		{
			Hello: &hellopb.Hello{
				FirstName: "Javier",
				Prefix:    "Sr",
			},
		},
		{
			Hello: &hellopb.Hello{
				FirstName: "Sandra",
				Prefix:    "Sra",
			},
		},
		{
			Hello: &hellopb.Hello{
				FirstName: "Jonathan",
				Prefix:    "Ing",
			},
		},
		{
			Hello: &hellopb.Hello{
				FirstName: "Sol",
				Prefix:    "Sra",
			},
		},
	}

	if err != nil {
		log.Fatalf("Error Creating Stream %v", err)
	}
	//Canal para esperar a que se complete todo y luego cerrar
	waitc := make(chan struct{})
	//enviar el mensaje al server (go routines)
	go func() {
		for _, req := range reqs {
			log.Printf("Enviando Mensaje!:: %v ", req)
			str.Send(req)
			time.Sleep(1000 * time.Millisecond)
		}
		str.CloseSend()
	}()

	//recibiendo mensajes del server (go routines)
	go func() {
		for {
			mess, err := str.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Error getting the response Server RPC %v", err)
				break
			}
			log.Printf("Got It!!!: %v", mess.GetGoodbye())
		}
		close(waitc)
	}()
	//block when everything is completed or closed
	<-waitc
}
