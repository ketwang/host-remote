package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestHostRemote(t *testing.T)  {
	RegisterFailHandler(Fail)
	RunSpecs(t, "plugins/ipam/host-remote")
}