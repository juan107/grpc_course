package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"hello/hellopb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type server struct{}

// Unary Server
func (*server) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	fmt.Println("Hello!, the function was called :)", req)

	firstName := req.GetHello().GetFirstName()
	prefix := req.GetHello().GetPrefix()

	customHello := "Welcome!!! " + prefix + " " + firstName
	resp := &hellopb.HelloResponse{
		CustomHello: customHello,
	}
	return resp, nil
}

// Server Streaming Server
func (*server) HelloManyLanguages(req *hellopb.HelloManyLanguagesRequest, stream hellopb.HelloService_HelloManyLanguagesServer) error {

	fmt.Println("Hello!, the server streaming was called :)", req)
	langs := [5]string{"Salut!", "Tonces", "Que Más", "Schalom", "Ni Hao!"}
	firstName := req.GetHello().GetFirstName()
	prefix := req.GetHello().GetPrefix()

	for _, lang := range langs {
		helloLang := lang + " " + prefix + " " + firstName
		res := &hellopb.HelloManyLanguagesResponse{
			HelloLanguage: helloLang,
		}
		stream.Send(res)
		time.Sleep(1000 * time.Millisecond) //one second
	}
	return nil
}

// Client Stream Server
func (*server) HellosGoodbye(stream hellopb.HelloService_HellosGoodbyeServer) error {
	fmt.Println("Hello!, the client streaming was called :)", stream)
	goodbye := "Chao Agonías: "
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			//Once the stream is finished, we send the response
			return stream.SendAndClose(&hellopb.HellosGoodbyeResponse{
				Goodbye: goodbye,
			})
		}
		if err != nil {
			log.Fatalf("Error getting data from stream %v", err)
		}
		firstName := req.GetHello().GetFirstName()
		prefix := req.GetHello().GetPrefix()
		goodbye += prefix + " " + firstName + " "
	}
}

// Bidirectional Stream Server
func (*server) Goodbye(stream hellopb.HelloService_GoodbyeServer) error {
	fmt.Println("Hello!, the bidirectional Server was called :)")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			//Once the stream is finished, we send the response
			return nil
		}
		if err != nil {
			log.Fatalf("Error getting data from stream %v", err)
			return nil
		}
		firstName := req.GetHello().GetFirstName()
		prefix := req.GetHello().GetPrefix()

		goodbye := prefix + "" + firstName + " "
		sendErr := stream.Send(&hellopb.GoodbyeResponse{
			Goodbye: goodbye,
		})

		if sendErr != nil {
			log.Fatalf("Error Sending Data to Client %v", sendErr)
		}
	}
}

func main() {
	fmt.Print("Hola, El servidor de Go está corriendo")

	cred, sslErr := credentials.NewServerTLSFromFile("../ssl/server.crt", "../ssl/server.key")
	if sslErr != nil {
		log.Fatalf("Failed to load certificate %v", sslErr)
	}

	lis, err := net.Listen("tcp", "0.0.0.0:50051")

	if err != nil {
		log.Fatalf("Failed to listen %v", err)
	}
	//Creacion de servidor con el certificado SSL
	opts := grpc.Creds(cred)
	s := grpc.NewServer(opts)
	hellopb.RegisterHelloServiceServer(s, &server{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to Serve %v", err)
	}

}
