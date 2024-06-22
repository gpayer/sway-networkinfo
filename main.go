package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/godbus/dbus/v5"
)

type ConnectionActive struct {
	Type      string
	State     uint32
	Default   bool
	Vpn       bool
	Ip4Config dbus.ObjectPath
	Devices   []dbus.ObjectPath
}

func main() {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		errExit(err)
	}

	nm := conn.Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager")

	var activeConnections []dbus.ObjectPath
	err = nm.StoreProperty("org.freedesktop.NetworkManager.ActiveConnections", &activeConnections)
	if err != nil {
		errExit(err)
	}

	// fmt.Printf("ActiveConnections: %v\n", activeConnections)

	for _, path := range activeConnections {
		ac, err := getActiveConnection(conn, path)
		if err != nil {
			errExit(err)
		}
		// fmt.Printf("ActiveConnection: %#v\n", ac)

		if ac.Default {
			addr, err := getIp4Config(conn, ac.Ip4Config)
			if err != nil {
				errExit(err)
			}
			iface, err := getDeviceInterface(conn, ac.Devices[0])
			if err != nil {
				errExit(err)
			}

			tx1, err := getTransferredBytes("tx", iface)
			if err != nil {
				errExit(err)
			}

			rx1, err := getTransferredBytes("rx", iface)
			if err != nil {
				errExit(err)
			}

			time.Sleep(1 * time.Second)

			tx2, err := getTransferredBytes("tx", iface)
			if err != nil {
				errExit(err)
			}

			rx2, err := getTransferredBytes("rx", iface)
			if err != nil {
				errExit(err)
			}

			tx := float64(tx2-tx1) / 1024.0
			txUnit := "KB/s"
			rx := float64(rx2-rx1) / 1024.0
			rxUnit := "KB/s"

			if tx > 1024 {
				tx = tx / 1024.0
				txUnit = "MB/s"
			}

			if rx > 1024 {
				rx = rx / 1024.0
				rxUnit = "MB/s"
			}

			extraTooltip := ""

			ifIcon := "󰛳"
			if ac.Type == "802-11-wireless" {
				ifIcon = ""
				ssid, err := getDeviceSsid(conn, ac.Devices[0])
				if err != nil {
					errExit(err)
				}
				addr = ssid
				extraTooltip = fmt.Sprintf(", SSID: %s", ssid)
			}

			out := map[string]interface{}{
				"text":    fmt.Sprintf("%.1f%s %.1f%s %s %s ", rx, rxUnit, tx, txUnit, addr, ifIcon),
				"tooltip": fmt.Sprintf("Interface: %s, Type: %s%s", iface, ac.Type, extraTooltip),
			}
			encJson, err := json.Marshal(out)
			if err != nil {
				errExit(err)
			}
			fmt.Println(string(encJson))
			return
		}
	}
	errExit(errors.New("no active connection found"))
}

func errExit(err error) {
	out := map[string]interface{}{
		"text":    "Error",
		"tooltip": err.Error(),
	}
	encJson, err := json.Marshal(out)
	if err != nil {
		out["tooltip"] = "Error while encoding error message"
	}
	fmt.Println(string(encJson))
	os.Exit(0)
}

func getActiveConnection(conn *dbus.Conn, acPath dbus.ObjectPath) (*ConnectionActive, error) {
	ac := conn.Object("org.freedesktop.NetworkManager", acPath)

	var state uint32
	err := ac.StoreProperty("org.freedesktop.NetworkManager.Connection.Active.State", &state)
	if err != nil {
		return nil, err
	}

	var ip4Config dbus.ObjectPath
	err = ac.StoreProperty("org.freedesktop.NetworkManager.Connection.Active.Ip4Config", &ip4Config)
	if err != nil {
		return nil, err
	}

	var connectionType string
	err = ac.StoreProperty("org.freedesktop.NetworkManager.Connection.Active.Type", &connectionType)
	if err != nil {
		return nil, err
	}

	var isDefault bool
	err = ac.StoreProperty("org.freedesktop.NetworkManager.Connection.Active.Default", &isDefault)
	if err != nil {
		return nil, err
	}

	var isVpn bool
	err = ac.StoreProperty("org.freedesktop.NetworkManager.Connection.Active.Vpn", &isVpn)
	if err != nil {
		return nil, err
	}

	var devices []dbus.ObjectPath
	err = ac.StoreProperty("org.freedesktop.NetworkManager.Connection.Active.Devices", &devices)
	if err != nil {
		return nil, err
	}

	return &ConnectionActive{
		Type:      connectionType,
		State:     state,
		Default:   isDefault,
		Vpn:       isVpn,
		Ip4Config: ip4Config,
		Devices:   devices,
	}, nil
}

func getIp4Config(conn *dbus.Conn, ip4config dbus.ObjectPath) (string, error) {
	ip4 := conn.Object("org.freedesktop.NetworkManager", ip4config)

	var addresses []map[string]dbus.Variant
	err := ip4.StoreProperty("org.freedesktop.NetworkManager.IP4Config.AddressData", &addresses)
	if err != nil {
		return "", err
	}

	if len(addresses) == 0 {
		return "", errors.New("no address data")
	}

	// addr := addresses[0]["address"].String() + "/" + addresses[0]["string"].String()
	address := addresses[0]
	// for k, v := range addresses[0] {
	// 	fmt.Printf("Key: %s, Value: %s\n", k, v.String())
	// }
	prefix, _ := address["prefix"].Value().(uint32)
	addr, _ := address["address"].Value().(string)
	return addr + "/" + strconv.Itoa(int(prefix)), nil
}

func getDeviceInterface(conn *dbus.Conn, devicePath dbus.ObjectPath) (string, error) {
	device := conn.Object("org.freedesktop.NetworkManager", devicePath)

	var iface string
	err := device.StoreProperty("org.freedesktop.NetworkManager.Device.Interface", &iface)
	if err != nil {
		return "", err
	}

	return iface, nil
}

func getDeviceSsid(conn *dbus.Conn, devicePath dbus.ObjectPath) (string, error) {
	device := conn.Object("org.freedesktop.NetworkManager", devicePath)

	var accessPoint dbus.ObjectPath
	err := device.StoreProperty("org.freedesktop.NetworkManager.Device.Wireless.ActiveAccessPoint", &accessPoint)
	if err != nil {
		return "", err
	}

	accessPointObj := conn.Object("org.freedesktop.NetworkManager", accessPoint)

	var ssid []byte
	err = accessPointObj.StoreProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid", &ssid)
	if err != nil {
		return "", err
	}

	return string(ssid), nil
}

func getTransferredBytes(typ string, dev string) (uint64, error) {
	raw, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/%s_bytes", dev, typ))
	if err != nil {
		return 0, err
	}
	if raw[len(raw)-1] == '\n' {
		raw = raw[:len(raw)-1]
	}

	return strconv.ParseUint(string(raw), 10, 64)
}
