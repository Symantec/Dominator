package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/srpc"
)

func changeImageExpirationSubcommand(args []string) {
	imageSClient, _ := getClients()
	err := changeImageExpiration(imageSClient, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error changing image expiration: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func changeImageExpiration(imageSClient *srpc.Client, name string) error {
	var expiresAt time.Time
	if *expiresIn > 0 {
		expiresAt = time.Now().Add(*expiresIn)
	}
	return client.ChangeImageExpiration(imageSClient, name, expiresAt)
}

func getImageExpirationSubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := getImageExpiration(imageSClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting image expiration: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getImageExpiration(imageSClient *srpc.Client, name string) error {
	expiresAt, err := client.GetImageExpiration(imageSClient, name)
	if err != nil {
		return err
	}
	if expiresAt.IsZero() {
		fmt.Println("image does not expire")
	} else if timeLeft := time.Until(expiresAt); timeLeft < 0 {
		fmt.Printf("image expired at %s (%s ago)\n", expiresAt,
			format.Duration(-timeLeft))
	} else {
		fmt.Printf("image expires at %s (in %s)\n", expiresAt,
			format.Duration(timeLeft))
	}
	return nil
}
