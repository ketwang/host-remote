package main

import (
	"encoding/json"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/testutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

var _ = Describe("host remote Operations", func() {
	It("allocates and releases address with ADD/DEL", func() {
		const ifname string = "eth0"
		const nspath string = "/path/to/ns"

		conf := `{
			"cniVersion": "0.3.1",
			"name": "blackFaceQuestion",
			"type": "macvlan",
			"master": "bond0",
			"ipam": {
				"type": "host-remote",
				"ipam_server": "http://127.0.0.1:5000"
			}
		}`

		args := &skel.CmdArgs{
			ContainerID: "fake_container_id",
			Netns:       nspath,
			IfName:      ifname,
			StdinData:   []byte(conf),
		}

		go inlineServer()
		//go server()

		// allocate ip
		r, raw, err := testutils.CmdAddWithArgs(args, func() error {
			return cmdAdd(args)
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(strings.Index(string(raw), "\"version\":")).Should(BeNumerically(">", 0))

		result, err := current.GetResult(r)
		Expect(err).NotTo(HaveOccurred())

		Expect(len(result.IPs)).To(Equal(1))

		Expect(*result.IPs[0]).To(Equal(current.IPConfig{
			Version: "4",
			Address: mustCIDR("10.35.4.1/20"),
			Gateway: mustGateWay("10.35.0.1"),
		}))

		// release ip
		err = testutils.CmdDelWithArgs(args, func() error {
			return cmdDel(args)
		})
		Expect(err).NotTo(HaveOccurred())
	})
})

func mustCIDR(s string) net.IPNet {
	ip, n, err := net.ParseCIDR(s)
	n.IP = ip
	if err != nil {
		Fail(err.Error())
	}

	return *n
}

func mustGateWay(s string) net.IP {
	return net.ParseIP(s)
}

type Item struct {
	Version     string `json:"version"`
	Address     string `json:"address"`
	Gateway     string `json:"gateway"`
	used        bool
	containerID string
}

func inlineServer() {
	http.HandleFunc("/register", func(writer http.ResponseWriter, request *http.Request) {
		ipAddr := Item{
			Version: "4",
			Address: "10.35.4.1/20",
			Gateway: "10.35.0.1",
			used:    false,
		}

		req := &PostBody{}
		content, err := ioutil.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(content, req)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		content, err = json.Marshal(ipAddr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		writer.WriteHeader(http.StatusOK)
		writer.Write(content)

	})

	http.HandleFunc("/unregister", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})

	http.ListenAndServe(":5000", nil)
}
