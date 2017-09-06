package main

import (
	"database/sql"
	"flag"
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

func runSQL(db *sql.DB, command string) {
	rows, err := db.Query(command)
	checkErr(err)
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error())
	}

	// Make a slice for the values
	values := make([]interface{}, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	fmt.Println("-----------------------------------")
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error())
		}

		// Print data
		for i, value := range values {
			switch value.(type) {
			case nil:
				fmt.Println(columns[i], ": NULL")

			case []byte:
				fmt.Println(columns[i], ": ", string(value.([]byte)))

			default:
				fmt.Println(columns[i], ": ", value)
			}
		}
		fmt.Println("-----------------------------------")
	}
}

func openDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./awsql.db")
	checkErr(err)
	sqlExec(db, `CREATE TABLE IF NOT EXISTS instances( 
		instance_id string, image_id  string,
		private_ip string, public_ip string,
		public_dnsname string, keyname string,
		name string, iam_profile string
		);
		
		DELETE FROM instances;
		VACUUM;
	`)

	return db
}

func main() {

	var query string
	flag.StringVar(&query, "q", "", "SQL Query")
	flag.Parse()
	// fmt.Println("Query: " + query)

	db := openDB()
	instancesToDB(db, listInstances())
	runSQL(db, query)
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
