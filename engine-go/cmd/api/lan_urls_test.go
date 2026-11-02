package main

import (
	"net"
	"slices"
	"testing"

	"t20engine/infra/config"
)

func address(cidr string) net.Addr {
	ip, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	network.IP = ip
	return network
}

// As interfaces do Docker não levam a lugar nenhum fora da máquina, e o deploy mostra o último endereço da
// lista: com elas nele, o deploy dizia um endereço que nenhum jogador consegue abrir.
func TestLanURLsListOnlyWhatPlayersCanOpen(t *testing.T) {
	got := lanURLsOf([]lanInterface{
		{Name: "lo", Addrs: []net.Addr{address("127.0.0.1/8")}},
		{Name: "enp0s20", Addrs: []net.Addr{address("192.168.1.36/24"), address("fe80::1/64")}},
		{Name: "wlp9s0", Addrs: []net.Addr{address("192.168.1.28/24")}},
		{Name: "docker0", Addrs: []net.Addr{address("172.17.0.1/16")}},
		{Name: "br-fd20797a19d0", Addrs: []net.Addr{address("172.19.0.1/16")}},
		{Name: "veth12ab", Addrs: []net.Addr{address("169.254.3.4/16")}},
	}, config.Config{Port: "3001"})
	want := []string{"http://192.168.1.36:3001", "http://192.168.1.28:3001"}
	if !slices.Equal(got, want) {
		t.Errorf("a lista é %v, e os jogadores só abrem %v", got, want)
	}
}
