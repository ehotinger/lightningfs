package mount

import (
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	fuseFS "bazil.org/fuse/fs"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/ehotinger/lightningfs/config"
	lightningFS "github.com/ehotinger/lightningfs/fs"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	defaultMntPoint = "/mnt/lightning"
)

// Command performs a mount.
var Command = cli.Command{
	Name:      "mount",
	Usage:     "perform a mount",
	ArgsUsage: "[mount]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug mode",
		},
		cli.StringFlag{
			Name:  "account-name",
			Usage: "Azure Blob account name",
		},
		cli.StringFlag{
			Name:  "account-key",
			Usage: "Azure Blob account key",
		},
		cli.StringFlag{
			Name:  "cache-path",
			Usage: "The location of the disk cache",
		},
		cli.StringFlag{
			Name:  "config-file",
			Usage: "The location of the configuration file",
		},
	},
	Action: func(context *cli.Context) error {
		var (
			mntPoint    = context.Args().First()
			debug       = context.Bool("debug")
			accountName = context.String("account-name")
			accountKey  = context.String("account-key")
			cachePath   = context.String("cache-path")
			configFile  = context.String("config-file")
		)
		config, err := config.NewConfigFromFile(configFile)
		if err != nil {
			return err
		}
		accountName = config.AzureAccountName
		accountKey = config.AzureAccountKey
		cachePath = config.CachePath

		_ = cachePath // TODO: implement caching

		if mntPoint == "" {
			mntPoint = defaultMntPoint
		}

		if accountName == "" {
			return errors.New("account name is required")
		}

		if accountKey == "" {
			return errors.New("account key is required")
		}

		// TODO: SAS support
		credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return errors.Wrap(err, "failed to create shared key credential")
		}

		if debug {
			fuse.Debug = func(msg interface{}) {
				log.Println(msg)
			}
		}

		ltFS, err := lightningFS.NewLightningFS(credential)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "Using %s as the mount point\n", mntPoint)

		c, err := fuse.Mount(mntPoint, fuse.FSName("ltfs"), fuse.Subtype("ltfs"), fuse.ReadOnly())
		if err != nil {
			log.Fatalf("failed to perform fuse mount, err: %v", err)
		}
		defer c.Close()
		defer fuse.Unmount(mntPoint)

		err = fuseFS.Serve(c, ltFS)
		if err != nil {
			return err
		}

		<-c.Ready
		return c.MountError
	},
}
