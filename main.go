package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"io/ioutil"
	"net"
	"net/http"
)

const (
	REGISTER   = "/register"
	UNREGISTER = "/unregister"
)

type Net struct {
	Name       string      `json:"name"`
	CNIVersion string      `json:"cniVersion"`
	IPAM       *IPAMConfig `json:"ipam"`
}

type IPAMConfig struct {
	Name       string
	Type       string `json:"type"`
	IPAMServer string `json:"ipam_server"`
}

type Address struct {
	AddressStr string `json:"address"`
	GatewayStr string `json:"gateway"`
	Version    string `json:"version"`
	Gateway    net.IP
	Address    net.IPNet
}

type PostBody struct {
	ContainerID string `json:"container_id"`
	Netns       string `json:"net_ns"`
	IfName      string `json:"if_name"`
	Args        string `json:"args"`
	Path        string `json:"path"`
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("host-remote"))
}

func loadIPAMConfig(bytes []byte) (*IPAMConfig, error) {
	n := &Net{}

	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("failed to load ipam config: %v", err)
	}

	return n.IPAM, nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func cmdAdd(args *skel.CmdArgs) error {
	versionDecoder := &version.ConfigDecoder{}
	confVersion, err := versionDecoder.Decode(args.StdinData)
	if err != nil {
		return err
	}

	ipAddr := &Address{}

	if err := handle(REGISTER, args, ipAddr); err != nil {
		return err
	}

	ip, n, err := net.ParseCIDR(ipAddr.AddressStr)
	if err != nil {
		return err
	}
	n.IP = ip
	ipAddr.Address = *n

	gw := net.ParseIP(ipAddr.GatewayStr)
	ipAddr.Gateway = gw

	result := &current.Result{}
	result.IPs = append(result.IPs, &current.IPConfig{
		Version: ipAddr.Version,
		Address: ipAddr.Address,
		Gateway: ipAddr.Gateway,
	})

	return types.PrintResult(result, confVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	return handle(UNREGISTER, args, nil)
}

func handle(action string, args *skel.CmdArgs, result *Address) error {
	suffix := REGISTER
	switch action {
	case REGISTER, UNREGISTER:
		suffix = action
	default:
		return fmt.Errorf("action must be in %s or %s", REGISTER, UNREGISTER)
	}

	ipamConfig, err := loadIPAMConfig(args.StdinData)
	if err != nil {
		return err
	}

	client := &http.Client{}

	kv := PostBody{
		ContainerID: args.ContainerID,
		Netns:       args.Netns,
		IfName:      args.IfName,
		Args:        args.Args,
		Path:        args.Path,
	}

	content, err := json.Marshal(kv)
	if err != nil {
		return err
	}

	body := bytes.NewBuffer(content)

	request, err := http.NewRequest(http.MethodPost, ipamConfig.IPAMServer+suffix, body)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	content, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("%s: %s", "ret code not 200", string(content))
	}

	if action == REGISTER {
		err := json.Unmarshal(content, result)
		if err != nil {
			return fmt.Errorf("%s", string(content))
		}
	}

	return nil
}
