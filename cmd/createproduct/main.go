package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/file"
	"github.com/stripe/stripe-go/v72/price"
	"github.com/stripe/stripe-go/v72/product"
)

const (
	// Price in cents
	Price = 6000
	// PriceCurrency pick from stripe list
	PriceCurrency = stripe.CurrencyCAD
	// ProductName is the name of the thing you want to sell
	ProductName = "Menu du 5 au 9 nov"
	// ProductDescription is the description of the thing you want to sell
	ProductDescription = ProductName
	// FilePath is the path of the image of the product you want to sell
	FilePath = "design/image.png"
)

type Config struct {
	StripeSecretKey string
}

func (c *Config) isTest() bool {
	if c.StripeSecretKey[0:8] == "sk_test_" {
		return true
	}
	return false
}

func envConfig() *Config {
	return &Config{
		StripeSecretKey: os.Getenv("STRIPE_SECRET_KEY"),
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	config := envConfig()
	stripe.Key = config.StripeSecretKey

	log.Println("Creating product")
	productParams := &stripe.ProductParams{
		Active:      stripe.Bool(true),
		Name:        stripe.String(ProductName),
		Description: stripe.String(ProductDescription),
		Type:        stripe.String(string(stripe.ProductTypeGood)),
	}

	newProduct, err := product.New(productParams)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Product '%v' created", newProduct.ID)

	log.Printf("Associating price to product '%v'", newProduct.ID)
	// Create the new product price
	newPrice, err := price.New(&stripe.PriceParams{
		Currency: stripe.String(string(PriceCurrency)),
		Product:  &newProduct.ID,
		// Base unit is cents
		UnitAmount: stripe.Int64(Price),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Price '%v' associated to product '%v'", newPrice.ID, newProduct.ID)

	log.Printf("Creating Product '%v' image", newProduct.ID)
	f, err := os.Open(FilePath)

	base := filepath.Base(FilePath)
	newFile, err := file.New(&stripe.FileParams{
		FileReader: f,
		Filename:   stripe.String(base),
		Purpose:    stripe.String("product_image"),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("File '%v' created now associating with product '%v'", newFile.ID, newProduct.ID)

	product.Update(newProduct.ID, &stripe.ProductParams{
		Images: []*string{stripe.String(newFile.URL)},
	})

	log.Printf("Go in stripe validate that everything is good %s", newProduct.URL)
}
