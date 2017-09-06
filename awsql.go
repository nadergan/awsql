package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	_ "github.com/mattn/go-sqlite3"
)

func listInstances() *ec2.DescribeInstancesOutput {
	ec2svc := ec2.New(session.New())
	resp, err := ec2svc.DescribeInstances(nil)
	if err != nil {
		fmt.Println("there was an error listing instances in", err.Error())
		log.Fatal(err.Error())
		return nil
	}
	return resp
}

func instancesToDB(db *sql.DB, instances *ec2.DescribeInstancesOutput) {
	for idx := range instances.Reservations {
		for _, inst := range instances.Reservations[idx].Instances {

			// Get iamProfile and beware of nil values.
			var iamProfile ec2.IamInstanceProfile
			if inst.IamInstanceProfile != nil {
				iamProfile = *inst.IamInstanceProfile
			}
			// Get instance Public IP address and beware of nils!
			var publicIP string
			if inst.PublicIpAddress != nil {
				publicIP = *inst.PublicIpAddress
			} else {
				publicIP = ""
			}
			//sqlExec(db, "DROP TABLE IF EXISTS instances")
			sqlExec(db, `CREATE TABLE IF NOT EXISTS instances( 
						instance_id string, image_id  string,
						private_ip string, public_ip string,
						public_dnsname string, keyname string,
						name string, iam_profile string
						);
					`)

			stmt, err := db.Prepare(`INSERT INTO instances(
															instance_id, image_id, private_ip,
															public_ip, public_dnsname, keyname,
															name, iam_profile
															)
									 values(?,?,?,?,?,?,?,?)`)
			checkErr(err)
			_, err = stmt.Exec(*inst.InstanceId, *inst.ImageId, *inst.PrivateIpAddress, publicIP,
				*inst.PublicDnsName, *inst.KeyName, *inst.Tags[0].Value, iamProfile.Arn)
			checkErr(err)
			// fmt.Println(*inst.InstanceId, *inst.ImageId, *inst.PrivateIpAddress, publicIP,
			// 	*inst.PublicDnsName, *inst.KeyName, *inst.Tags[0].Value, iamProfile.Arn)
		}
	}

}

func openDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./aws.db")
	checkErr(err)
	return db
}

func main() {
	db := openDB()
	instancesToDB(db, listInstances())
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func sqlExec(db *sql.DB, command string) {
	_, err := db.Exec(command)
	checkErr(err)
}
