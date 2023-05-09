package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"products/productpb"

	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Go Clien Is Running")
	cc, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to Connect %v", err)
	}
	defer cc.Close()
	c := productpb.NewProductServiceClient(cc)

	fmt.Println("----|| CREATING PRODUCT ||----")
	product := &productpb.Product{
		Name:  "Play Station 5",
		Price: 5250000.50,
	}
	res, err := c.CreateProduct(context.Background(), &productpb.CreateProductRequest{
		Product: product,
	})
	if err != nil {
		log.Fatalf("Failed to Create Product %v", err)
	}
	fmt.Println("Product Created!!::: ", res)

	fmt.Println("----|| GETTING PRODUCT ||----")

	productId := res.GetProduct().GetId()
	getProductReq := &productpb.GetProductRequest{ProductId: productId}
	getProductRes, getProductErr := c.GetProduct(context.Background(), getProductReq)
	if getProductErr != nil {
		log.Fatalf("Failed to Get Product %v", getProductErr)
	}
	fmt.Println("Product Obtained!!::: ", getProductRes)

	fmt.Println("----|| UPDATING PRODUCT ||----")

	updateProduct := &productpb.Product{
		Id:    productId,
		Name:  "Iphone XXX",
		Price: 4540000.33,
	}
	updateRes, updateErr := c.UpdateProduct(context.Background(), &productpb.UpdateProductRequest{Product: updateProduct})
	if updateErr != nil {
		log.Fatalf("Failed to Update Product %v", updateErr)
	}
	fmt.Println("Product Updated!!::: ", updateRes)

	fmt.Println("----|| DELETED PRODUCT ||----")

	// deleteRes, deleteErr := c.DeleteProduct(context.Background(), &productpb.DeleteProductRequest{ProductId: productId})
	// if deleteErr != nil {
	// 	log.Fatalf("Failed to Delete Product %v", deleteErr)
	// }
	// fmt.Println("Product Deleted!!::: ", deleteRes.GetProductId())

	fmt.Println("----|| GETTING PRODUCT LIST ||----")

	stream, productListErr := c.GetProductList(context.Background(), &productpb.GetProductListRequest{})
	if productListErr != nil {
		log.Fatalf("Failed to Get Product List %v", productListErr)
	}
	for {
		str, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("Error recieving Products", err)
		}
		fmt.Println("Product Recieved!!!::: ", str.GetProduct())
	}
}
