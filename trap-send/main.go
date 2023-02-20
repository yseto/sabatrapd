package main

import (
	"log"

	g "github.com/gosnmp/gosnmp"
)

func main() {
	g.Default.Target = "127.0.0.1"
	g.Default.Port = 9162
	g.Default.Version = g.Version2c
	g.Default.Community = "public"
	// g.Default.Logger = g.NewLogger(log.New(os.Stdout, "", 0))
	err := g.Default.Connect()
	if err != nil {
		log.Fatalf("Connect() err: %v", err)
	}
	defer g.Default.Conn.Close()

	trap := g.SnmpTrap{
		Variables: []g.SnmpPDU{
			{Value: ".1.3.6.1.6.3.1.1.5.3", Name: ".1.3.6.1.6.3.1.1.4.1.0", Type: g.ObjectIdentifier},
			{Value: 9, Name: ".1.3.6.1.2.1.2.2.1.1.9", Type: g.Integer},
			{Value: []byte("dum0"), Name: ".1.3.6.1.2.1.2.2.1.2.9", Type: g.OctetString},
			{Value: 6, Name: ".1.3.6.1.2.1.2.2.1.3.9", Type: g.Integer},
			{Value: 2, Name: ".1.3.6.1.2.1.2.2.1.7.9", Type: g.Integer},
			{Value: 2, Name: ".1.3.6.1.2.1.2.2.1.8.9", Type: g.Integer},
			{Value: ".1.3.6.1.4.1.8072.3.2.10", Name: ".1.3.6.1.6.3.1.1.4.3.0", Type: g.ObjectIdentifier},
		},
	}
	/*
		trap := g.SnmpTrap{
			Variables: []g.SnmpPDU{
				//			{Name: ".1.3.6.1.6.3.1.1.4.1.0", Type: g.ObjectIdentifier, Value: ".1.3.6.1.6.3.1.1.5.1"},
				{Name: ".1.3.6.1.6.3.1.1.4.1.0", Type: g.ObjectIdentifier, Value: ".1.3.6.1.6.3.1.1.5.2"},
			},
		}
	*/

	/*
		trap := g.SnmpTrap{
			Variables: []g.SnmpPDU{
				{Value: ".1.3.6.1.4.1.9.9.43.2.0.1", Name: ".1.3.6.1.6.3.1.1.4.1.0", Type: g.ObjectIdentifier},
				{Value: 1, Name: ".1.3.6.1.4.1.9.9.43.1.1.6.1.3.10", Type: g.Integer},
				{Value: 2, Name: ".1.3.6.1.4.1.9.9.43.1.1.6.1.4.10", Type: g.Integer},
				{Value: 3, Name: ".1.3.6.1.4.1.9.9.43.1.1.6.1.5.10", Type: g.Integer},
			},
		}
	*/

	/*
		trap := g.SnmpTrap{
			Variables: []g.SnmpPDU{
				{Value: ".1.3.6.1.4.1.9.9.41.2.0.1", Name: ".1.3.6.1.6.3.1.1.4.1.0", Type: g.ObjectIdentifier},
				{Value: []byte("DOT11"), Name: ".1.3.6.1.4.1.9.9.41.1.2.3.1.2.1414", Type: g.OctetString},
				{Value: 5, Name: ".1.3.6.1.4.1.9.9.41.1.2.3.1.3.1414", Type: g.Integer},
				{Value: []byte("MAXRETRIES"), Name: ".1.3.6.1.4.1.9.9.41.1.2.3.1.4.1414", Type: g.OctetString},
				{Value: []byte("Packet to client e28c.c759.c887 reached max retries, removing the client"), Name: ".1.3.6.1.4.1.9.9.41.1.2.3.1.5.1414", Type: g.OctetString},
				{Value: uint32(79758314), Name: ".1.3.6.1.4.1.9.9.41.1.2.3.1.6.1414", Type: g.TimeTicks},
			},
		}
	*/

	_, err = g.Default.SendTrap(trap)
	if err != nil {
		log.Fatalf("SendTrap() err: %v", err)
	}
}
