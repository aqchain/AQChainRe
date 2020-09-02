package submodule

import (
	"AQChainRe/pkg/config"
	"context"
	"fmt"
	ds "github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"testing"
)

func TestNewNetworkSubmodule(t *testing.T) {

	logging.SetAllLoggers(logging.LevelInfo)
	ctx:=context.Background()

	cfg:= config.NewDefaultConfig()
	repo:=NetworkRepo{
		Config: nil,
		Datastore: ds.NewMapDatastore(),
	}
	ncfg:= NetworkConfig{
		OfflineMode: false,
		IsRelay:     false,
		Libp2pOpts:  nil,
	}
	networkSubmodule, err := NewNetworkSubmodule(context.Background(), ncfg , repo)

	if err!=nil{
		fmt.Println(err)
	}

	discoverySubmodule, err := NewDiscoverySubmodule(ctx, cfg.Bootstrap, &networkSubmodule)

	if err!=nil{
		fmt.Println(err)
	}
	peer1 := fmt.Sprintf("%s/ipfs/%s", networkSubmodule.Host.Addrs()[0].String(), networkSubmodule.Host.ID().Pretty())
	fmt.Println(peer1)

	err = discoverySubmodule.Start()
	if err!=nil{
		fmt.Println(err)
	}

	select {

	}
}

func TestNewNetworkSubmodule2(t *testing.T) {

	logging.SetAllLoggers(logging.LevelInfo)
	ctx:=context.Background()

	cfg:= config.NewDefaultConfig()
	cfg.Bootstrap.Addresses = []string{"/ip4/127.0.0.1/tcp/44161/ipfs/QmNVUa2Xq8Sg24a3zqcWv2hQZqx8hM7u8KyGyF8v1KRztv"}
	repo:=NetworkRepo{
		Config: nil,
		Datastore: ds.NewMapDatastore(),
	}
	ncfg:= NetworkConfig{
		OfflineMode: false,
		IsRelay:     false,
		Libp2pOpts:  nil,
	}
	networkSubmodule, err := NewNetworkSubmodule(context.Background(), ncfg , repo)

	if err!=nil{
		fmt.Println(err)
	}

	discoverySubmodule, err := NewDiscoverySubmodule(ctx, cfg.Bootstrap, &networkSubmodule)

	if err!=nil{
		fmt.Println(err)
	}
	fmt.Println(networkSubmodule.Host.ID())
	fmt.Println(networkSubmodule.Host.Addrs())

	err = discoverySubmodule.Start()
	if err!=nil{
		fmt.Println(err)
	}

	select {

	}
}
