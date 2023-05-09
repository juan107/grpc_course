package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"products/productpb"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct{}

var collection *mongo.Collection

type product struct {
	Id    primitive.ObjectID `bson:"_id,omitempty"`
	Name  string             `bson:"name,omitempty"`
	Price float64            `bson:"price"`
}

func (*server) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	productId := req.GetProductId()
	oid, err := primitive.ObjectIDFromHex(productId)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Internal Error Parsing Product Id: %v", err),
		)
	}
	data := &product{}
	filter := bson.M{"_id": oid}
	res := collection.FindOne(context.Background(), filter)

	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Internal Error Finding the Product: %v", err),
		)
	}
	return &productpb.GetProductResponse{
		Product: dbToProductPB(data),
	}, nil
}

func (*server) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	//Parse contenido y guardar en mongo
	prod := req.GetProduct()
	data := product{
		Name:  prod.GetName(),
		Price: prod.GetPrice(),
	}
	res, err := collection.InsertOne(context.Background(), data)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal Error: %v", err),
		)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal Error: %v", err),
		)
	}
	return &productpb.CreateProductResponse{
		Product: &productpb.Product{
			Id:    oid.Hex(),
			Name:  prod.GetName(),
			Price: prod.GetPrice(),
		},
	}, nil
}

func (*server) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	fmt.Println("Updating Product")
	prod := req.GetProduct()
	oid, err := primitive.ObjectIDFromHex(prod.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Error Parsing ProductId: %v", err),
		)
	}
	//Create empty struct
	data := &product{}
	filter := bson.M{"_id": oid}
	//Finding Product in DB
	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Error Finding Product For Update: %v", err),
		)
	}

	//update internal struct product
	data.Name = prod.GetName()
	data.Price = prod.GetPrice()

	//update database
	_, updErr := collection.ReplaceOne(context.Background(), filter, data)
	if updErr != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Error Updating Product in Database: %v", updErr),
		)
	}
	return &productpb.UpdateProductResponse{
		Product: dbToProductPB(data),
	}, nil
}

func (*server) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	fmt.Println("Deleting Product")
	productId := req.GetProductId()
	oid, err := primitive.ObjectIDFromHex(productId)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Internal Error Parsing Product Id: %v", err),
		)
	}
	filter := bson.M{"_id": oid}
	res, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Internal Error Deleting Product Id: %v", err),
		)
	}
	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			"No Product Was Deleted",
		)
	}
	return &productpb.DeleteProductResponse{
		ProductId: productId,
	}, nil
}

func (*server) GetProductList(req *productpb.GetProductListRequest, stream productpb.ProductService_GetProductListServer) error {
	fmt.Println("Getting Product List")
	cur, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal Error Getting Product List: %v", err),
		)
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		data := &product{}
		err := cur.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error Decoding Data: %v", err),
			)
		}
		stream.Send(&productpb.GetProductListResponse{
			Product: dbToProductPB(data),
		})
	}
	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Error Looping Data: %v", err),
		)
	}
	return nil
}

func main() {

	//Bandera para agregar mas detalle al error
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	client := connectToDataBase()
	fmt.Println("Hola, El servidor de Go está corriendo")
	lis, err := net.Listen("tcp", "0.0.0.0:50051")

	if err != nil {
		log.Fatalf("Failed to listen %v", err)
	}
	s := grpc.NewServer()
	productpb.RegisterProductServiceServer(s, &server{})

	go func() {
		fmt.Println("Starting Server")
		if err := s.Serve(lis); err != nil {
			log.Fatal("Failed to Serve:: ", err)
		}
	}()
	chn := make(chan os.Signal, 1)
	signal.Notify(chn, os.Interrupt)

	//Block hasta que se obtenga una señal
	<-chn
	fmt.Println("Stop Using the Server")
	s.Stop()
	fmt.Println("Closing the listener")
	lis.Close()
	fmt.Println("Closing DB Connection")
	client.Disconnect(context.TODO())
	fmt.Println("Goodbye :) ....")

}

func connectToDataBase() *mongo.Client {
	fmt.Println("Connecting to Mongo")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Failed to Connect to Mongo %v", err)
	}
	collection = client.Database("productdb").Collection("products")
	return client
}

func dbToProductPB(data *product) *productpb.Product {
	return &productpb.Product{
		Id:    data.Id.Hex(),
		Name:  data.Name,
		Price: data.Price,
	}
}
