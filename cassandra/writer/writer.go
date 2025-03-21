package main

import (
	"log"
	"time"

	"github.com/gocql/gocql"
)

func main() {
	cluster := gocql.NewCluster("PublicIP", "PublicIP", "PublicIP") //replace PublicIP with the IP addresses used by your cluster.
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = time.Second * 10
	cluster.Authenticator = gocql.PasswordAuthenticator{Username: "Username", Password: "Password", AllowedAuthenticators: []string{"com.instaclustr.cassandra.auth.InstaclustrPasswordAuthenticator"}} //replace the username and password fields with their real settings, you will need to allow the use of the Instaclustr Password Authenticator.
	session, err := cluster.CreateSession()
	if err != nil {
		log.Println(err)
		return
	}
	defer session.Close()

}
