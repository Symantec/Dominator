package main

import (
	"fmt"
	"time"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func changeImageExpirationSubcommand(args []string,
	logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	err := changeImageExpiration(imageSClient, args[0])
	if err != nil {
		return fmt.Errorf("Error changing image expiration: %s", err)
	}
	return nil
}

func changeImageExpiration(imageSClient *srpc.Client, name string) error {
	var expiresAt time.Time
	if *expiresIn > 0 {
		expiresAt = time.Now().Add(*expiresIn)
	}
	return client.ChangeImageExpiration(imageSClient, name, expiresAt)
}

func getImageExpirationSubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	if err := getImageExpiration(imageSClient, args[0]); err != nil {
		return fmt.Errorf("Error getting image expiration: %s", err)
	}
	return nil
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
