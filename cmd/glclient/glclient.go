// Copyright 2020 Demian Harvill
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command line client for gRPC MServiceLedger service
package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gaterace/dml-go/pkg/dml"

	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"regexp"
	"strconv"

	pb "github.com/gaterace/mledger/pkg/mserviceledger"
	"github.com/kylelemons/go-gypsy/yaml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"

	flag "github.com/juju/gnuflag"
)

var InvalidParameter = errors.New("invalid parameter")

var dateValidator = regexp.MustCompile("^\\d\\d\\d\\d\\-\\d\\d\\-\\d\\d$")
var idlistValidator = regexp.MustCompile("^\\d+(,\\d+)*$")
var validDecimal = regexp.MustCompile("^(\\+-)?\\d+(.\\d+)$")

var name = flag.String("name", "", "name")
var description = flag.String("desc", "", "description")
var sdate = flag.String("sdate", "", "start date")
var edate = flag.String("edate", "", "end date")
var guid = flag.String("guid", "", "guid")
var orgid = flag.String("orgid", "", "orgid")
var version = flag.Int64("version", -1, "version")
var id = flag.Int64("id", 0, "id")
var type_name = flag.String("type", "", "type")
var type_id = flag.Int64("type_id", 0, "type id")
var tdate = flag.String("tdate", "", "transaction date")
var from_party = flag.Int64("from_party", -1, "from party id")
var to_party = flag.Int64("to_party", -1, "to party id")
var via_key = flag.String("via_key", "", "posted via key")
var via_date = flag.String("via_date", "", "posted via date")
var json_str = flag.String("json", "", "transaction details as json")

func main() {
	flag.Parse(true)

	configFilename := "conf.yaml"
	usr, err := user.Current()
	if err == nil {
		homeDir := usr.HomeDir
		configFilename = homeDir + string(os.PathSeparator) + ".mledger.config"
	}

	config, err := yaml.ReadFile(configFilename)
	if err != nil {
		log.Fatalf("configuration not found: " + configFilename)
	}

	// log_file, _ := config.Get("log_file")
	ca_file, _ := config.Get("ca_file")
	tls, _ := config.GetBool("tls")
	server_host_override, _ := config.Get("server_host_override")
	server, _ := config.Get("server")
	port, _ := config.GetInt("port")

	if port == 0 {
		port = 50054
	}

	if len(flag.Args()) < 1 {
		prog := os.Args[0]
		fmt.Printf("Command line client for mledger grpc service\n")
		fmt.Printf("usage:\n")
		fmt.Printf("    %s create_organization --name <name> --sdate <start_date> [--edate <end_date>]\n", prog)
		fmt.Printf("    %s update_organization --orgid <orgid> --version <version> --name <name> --sdate <start_date> [--edate <end_date>]\n", prog)
		fmt.Printf("    %s delete_organization --orgid <orgid> --version <version>\n", prog)
		fmt.Printf("    %s get_organization_by_id --orgid <orgid> \n", prog)
		fmt.Printf("    %s get_organizations_by_mservice \n", prog)
		fmt.Printf("    %s create_account_type --id <id> --type <type> \n", prog)
		fmt.Printf("    %s update_account_type --id <id> --version <version> --type <type>\n", prog)
		fmt.Printf("    %s delete_account_type --id <id> --version <version>\n", prog)
		fmt.Printf("    %s get_account_type_by_id --id <id>\n", prog)
		fmt.Printf("    %s get_account_types_by_mservice\n", prog)
		fmt.Printf("    %s create_transaction_type --id <id> --type <type> \n", prog)
		fmt.Printf("    %s update_transaction_type --id <id>  --version <version>  --type <type> \n", prog)
		fmt.Printf("    %s delete_transaction_type --id <id>  --version <version>\n", prog)
		fmt.Printf("    %s get_transaction_type_by_id --id <id>\n", prog)
		fmt.Printf("    %s get_transaction_types_by_mservice\n", prog)
		fmt.Printf("    %s create_party --id <id> --name <name>\n", prog)
		fmt.Printf("    %s update_party --id <id> --name <name> --version <version>\n", prog)
		fmt.Printf("    %s delete_party --id <id> --version <version>\n", prog)
		fmt.Printf("    %s get_party_by_id --id <id> \n", prog)
		fmt.Printf("    %s get_parties_by_mservice\n", prog)
		fmt.Printf("    %s create_account --orgid <orgid> --name <name> --desc <description> --type_id <type_id>\n", prog)
		fmt.Printf("    %s update_account --guid <guid> --version <version> --name <name> --desc <description> --type_id <type_id>\n", prog)
		fmt.Printf("    %s delete_account --guid <guid> --version <version>\n", prog)
		fmt.Printf("    %s get_account_by_id --guid <guid>\n", prog)
		fmt.Printf("    %s get_accounts_by_organization --orgid <orgid>\n", prog)
		fmt.Printf("    %s create_transaction --orgid <orgid> --tdate <tdate> --desc <description> --type_id <type_id> [--from_party <from_party>] \n", prog)
		fmt.Printf("                  [--to_party <to_party> ] [--via_key <via_key> --via_date <via_date>]\n")
		fmt.Printf("    %s update_transaction --id <id>  --version <version>  --tdate <tdate> --desc <description> --type_id <type_id> [--from_party <from_party>] \n", prog)
		fmt.Printf("                  [--to_party <to_party> ] [--via_key <via_key> --via_date <via_date>]\n")
		fmt.Printf("    %s delete_transaction --id <id> --version <version>\n", prog)
		fmt.Printf("    %s get_transaction_by_id --id <id> \n", prog)
		fmt.Printf("    %s get_transaction_wrapper_by_id --id <id> \n", prog)
		fmt.Printf("    %s get_transaction_wrappers_by_date  --orgid <orgid> --sdate <start_date> --edate <end_date>\n", prog)
		fmt.Printf("    %s add_transaction_details --id <id> --json <json>\n", prog)
		fmt.Println("    example: --json '[{\"aid\": \"0123456789abcdef0123456789abcdef\", \"amt\": \"10.00\", \"debit\": true}, [\"aid\": \"3210456789abcdef0123456789abcdef\", \"amt\":\"10.00\"}]'")

		fmt.Printf("    %s get_server_version \n", prog)

		os.Exit(1)
	}

	cmd := flag.Arg(0)

	validParams := true
	var start_date *dml.DateTime
	var end_date *dml.DateTime
	var transaction_date *dml.DateTime
	var v_date *dml.DateTime
	var organization_id *dml.Guid
	var account_id *dml.Guid
	var details []*pb.GLTransactionDetail

	switch cmd {
	case "create_organization":
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
		date := *sdate
		if !dateValidator.MatchString(date) {
			fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		start_date = dml.DateTimeFromString(date)

		date = *edate
		if date != "" {
			if dateValidator.MatchString(date) {
				end_date = dml.DateTimeFromString(date)
			} else {
				fmt.Println("end_date parameter not in yyyy-mm-dd format")
				validParams = false
			}
		}

	case "update_organization":
		organization_id, err = dml.GuidFromString(*orgid)
		if err != nil {
			fmt.Println("orgid parameter missing or invalid")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
		date := *sdate
		if !dateValidator.MatchString(date) {
			fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		start_date = dml.DateTimeFromString(date)

		date = *edate
		if date != "" {
			if dateValidator.MatchString(date) {
				end_date = dml.DateTimeFromString(date)
			} else {
				fmt.Println("end_date parameter not in yyyy-mm-dd format")
				validParams = false
			}
		}

		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "delete_organization":
		organization_id, err = dml.GuidFromString(*orgid)
		if err != nil {
			fmt.Println("orgid parameter missing or invalid")
			validParams = false
		}

		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_organization_by_id":
		organization_id, err = dml.GuidFromString(*orgid)
		if err != nil {
			fmt.Println("orgid parameter missing or invalid")
			validParams = false
		}
	case "get_organizations_by_mservice":
		// no parameters
		validParams = true
	case "create_account_type":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *type_name == "" {
			fmt.Println("type parameter missing or invalid")
			validParams = false
		}

	case "update_account_type":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *type_name == "" {
			fmt.Println("type parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "delete_account_type":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_account_type_by_id":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
	case "get_account_types_by_mservice":
		// no parameters
		validParams = true
	case "create_transaction_type":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *type_name == "" {
			fmt.Println("type parameter missing or invalid")
			validParams = false
		}
	case "update_transaction_type":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *type_name == "" {
			fmt.Println("type parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "delete_transaction_type":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_transaction_type_by_id":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
	case "get_transaction_types_by_mservice":
		// no parameters
		validParams = true
	case "create_party":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing or invalid")
			validParams = false
		}
	case "update_party":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "delete_party":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_party_by_id":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
	case "get_parties_by_mservice":
		// no parameters
		validParams = true
	case "create_account":
		if *type_id <= 0 {
			fmt.Println("type_id parameter missing or invalid")
			validParams = false
		}
		if *name == "" {
			fmt.Println("type parameter missing or invalid")
			validParams = false
		}
		if *description == "" {
			fmt.Println("desc parameter missing or invalid")
			validParams = false
		}
		organization_id, err = dml.GuidFromString(*orgid)
		if err != nil {
			fmt.Println("orgid parameter missing or invalid")
			validParams = false
		}
	case "update_account":
		account_id, err = dml.GuidFromString(*guid)
		if err != nil {
			fmt.Println("guid parameter missing or invalid")
			validParams = false
		}

		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
		if *type_id <= 0 {
			fmt.Println("type_id parameter missing or invalid")
			validParams = false
		}
		if *name == "" {
			fmt.Println("type parameter missing or invalid")
			validParams = false
		}
		if *description == "" {
			fmt.Println("desc parameter missing or invalid")
			validParams = false
		}
	case "delete_account":
		account_id, err = dml.GuidFromString(*guid)
		if err != nil {
			fmt.Println("guid parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_account_by_id":
		account_id, err = dml.GuidFromString(*guid)
		if err != nil {
			fmt.Println("guid parameter missing or invalid")
			validParams = false
		}
	case "get_accounts_by_organization":
		organization_id, err = dml.GuidFromString(*orgid)
		if err != nil {
			fmt.Println("orgid parameter missing or invalid")
			validParams = false
		}

	case "create_transaction":
		organization_id, err = dml.GuidFromString(*orgid)
		if err != nil {
			fmt.Println("orgid parameter missing or invalid")
			validParams = false
		}
		date := *tdate
		if !dateValidator.MatchString(date) {
			fmt.Println("transaction_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		transaction_date = dml.DateTimeFromString(date)

		if *description == "" {
			fmt.Println("desc parameter missing or invalid")
			validParams = false
		}

		if *type_id <= 0 {
			fmt.Println("type_id parameter missing or invalid")
			validParams = false
		}

		date = *via_date
		if date != "" {
			if dateValidator.MatchString(date) {
				v_date = dml.DateTimeFromString(date)
			} else {
				fmt.Println("via_date parameter not in yyyy-mm-dd format")
				validParams = false
			}
		}

	case "update_transaction":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
		date := *tdate
		if !dateValidator.MatchString(date) {
			fmt.Println("transaction_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		transaction_date = dml.DateTimeFromString(date)

		if *description == "" {
			fmt.Println("desc parameter missing or invalid")
			validParams = false
		}

		if *type_id <= 0 {
			fmt.Println("type_id parameter missing or invalid")
			validParams = false
		}

		date = *via_date
		if date != "" {
			if dateValidator.MatchString(date) {
				v_date = dml.DateTimeFromString(date)
			} else {
				fmt.Println("via_date parameter not in yyyy-mm-dd format")
				validParams = false
			}
		}
	case "delete_transaction":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_transaction_by_id":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
	case "get_transaction_wrapper_by_id":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
	case "get_transaction_wrappers_by_date":
		organization_id, err = dml.GuidFromString(*orgid)
		if err != nil {
			fmt.Println("orgid parameter missing or invalid")
			validParams = false
		}

		date := *sdate
		if !dateValidator.MatchString(date) {
			fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		start_date = dml.DateTimeFromString(date)

		date = *edate
		if !dateValidator.MatchString(date) {
			fmt.Println("end_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		end_date = dml.DateTimeFromString(date)

	case "add_transaction_details":
		if *id <= 0 {
			fmt.Println("id parameter missing or invalid")
			validParams = false
		}
		details, err = TransformDetails(*id, *json_str)
		if err != nil {
			validParams = false
		}
		fmt.Printf("details: %v\n", details)
	case "get_server_version":
		validParams = true

	default:
		fmt.Printf("unknown command: %s\n", cmd)
		validParams = false
	}

	if !validParams {
		os.Exit(1)
	}

	tokenFilename := "token.txt"
	usr, err = user.Current()
	if err == nil {
		homeDir := usr.HomeDir
		tokenFilename = homeDir + string(os.PathSeparator) + ".mservice.token"
	}

	address := server + ":" + strconv.Itoa(int(port))

	var opts []grpc.DialOption
	if tls {
		var sn string
		if server_host_override != "" {
			sn = server_host_override
		}
		var creds credentials.TransportCredentials
		if ca_file != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(ca_file, sn)
			if err != nil {
				grpclog.Fatalf("Failed to create TLS credentials %v", err)
			}
		} else {
			creds = credentials.NewClientTLSFromCert(nil, sn)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// set up connection to server
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	client := pb.NewMServiceLedgerClient(conn)

	ctx := context.Background()

	savedToken := ""

	data, err := ioutil.ReadFile(tokenFilename)

	if err == nil {
		savedToken = string(data)
	}

	md := metadata.Pairs("token", savedToken)
	mctx := metadata.NewOutgoingContext(ctx, md)

	switch cmd {
	case "create_organization":
		req := pb.CreateOrganizationRequest{}
		req.OrganizationName = *name
		req.FromDate = start_date
		if *edate != "" {
			req.ToDate = end_date
		}

		resp, err := client.CreateOrganization(mctx, &req)
		printResponse(resp, err)

	case "update_organization":
		req := pb.UpdateOrganizationRequest{}
		req.OrganizationId = organization_id
		req.Version = int32(*version)
		req.OrganizationName = *name
		req.FromDate = start_date
		if *edate != "" {
			req.ToDate = end_date
		}

		resp, err := client.UpdateOrganization(mctx, &req)
		printResponse(resp, err)

	case "delete_organization":
		req := pb.DeleteOrganizationRequest{}
		req.OrganizationId = organization_id
		req.Version = int32(*version)

		resp, err := client.DeleteOrganization(mctx, &req)
		printResponse(resp, err)

	case "get_organization_by_id":
		req := pb.GetOrganizationByIdRequest{}
		req.OrganizationId = organization_id
		resp, err := client.GetOrganizationById(mctx, &req)
		printResponse(resp, err)

	case "get_organizations_by_mservice":
		req := pb.GetOrganizationsByMserviceRequest{}
		resp, err := client.GetOrganizationsByMservice(mctx, &req)
		printResponse(resp, err)
	case "create_account_type":
		req := pb.CreateAccountTypeRequest{}
		req.AccountTypeId = int32(*id)
		req.AccountType = *type_name
		resp, err := client.CreateAccountType(mctx, &req)
		printResponse(resp, err)
	case "update_account_type":
		req := pb.UpdateAccountTypeRequest{}
		req.AccountTypeId = int32(*id)
		req.AccountType = *type_name
		req.Version = int32(*version)
		resp, err := client.UpdateAccountType(mctx, &req)
		printResponse(resp, err)
	case "delete_account_type":
		req := pb.DeleteAccountTypeRequest{}
		req.AccountTypeId = int32(*id)
		req.Version = int32(*version)
		resp, err := client.DeleteAccountType(mctx, &req)
		printResponse(resp, err)
	case "get_account_type_by_id":
		req := pb.GetAccountTypeByIdRequest{}
		req.AccountTypeId = int32(*id)
		resp, err := client.GetAccountTypeById(mctx, &req)
		printResponse(resp, err)
	case "get_account_types_by_mservice":
		req := pb.GetAccountTypesByMserviceRequest{}
		resp, err := client.GetAccountTypesByMservice(mctx, &req)
		printResponse(resp, err)
	case "create_transaction_type":
		req := pb.CreateTransactionTypeRequest{}
		req.TransactionTypeId = int32(*id)
		req.TransactionType = *type_name
		resp, err := client.CreateTransactionType(mctx, &req)
		printResponse(resp, err)
	case "update_transaction_type":
		req := pb.UpdateTransactionTypeRequest{}
		req.TransactionTypeId = int32(*id)
		req.TransactionType = *type_name
		req.Version = int32(*version)
		resp, err := client.UpdateTransactionType(mctx, &req)
		printResponse(resp, err)
	case "delete_transaction_type":
		req := pb.DeleteTransactionTypeRequest{}
		req.TransactionTypeId = int32(*id)
		req.Version = int32(*version)
		resp, err := client.DeleteTransactionType(mctx, &req)
		printResponse(resp, err)
	case "get_transaction_type_by_id":
		req := pb.GetTransactionTypeByIdRequest{}
		req.TransactionTypeId = int32(*id)
		resp, err := client.GetTransactionTypeById(mctx, &req)
		printResponse(resp, err)
	case "get_transaction_types_by_mservice":
		req := pb.GetTransactionTypesByMserviceRequest{}
		resp, err := client.GetTransactionTypesByMservice(mctx, &req)
		printResponse(resp, err)
	case "create_party":
		req := pb.CreatePartyRequest{}
		req.PartyId = *id
		req.PartyName = *name
		resp, err := client.CreateParty(mctx, &req)
		printResponse(resp, err)
	case "update_party":
		req := pb.UpdatePartyRequest{}
		req.PartyId = *id
		req.PartyName = *name
		req.Version = int32(*version)
		resp, err := client.UpdateParty(mctx, &req)
		printResponse(resp, err)
	case "delete_party":
		req := pb.DeletePartyRequest{}
		req.PartyId = *id
		req.Version = int32(*version)
		resp, err := client.DeleteParty(mctx, &req)
		printResponse(resp, err)
	case "get_party_by_id":
		req := pb.GetPartyByIdRequest{}
		req.PartyId = *id
		resp, err := client.GetPartyById(mctx, &req)
		printResponse(resp, err)
	case "get_parties_by_mservice":
		req := pb.GetPartiesByMserviceRequest{}
		resp, err := client.GetPartiesByMservice(mctx, &req)
		printResponse(resp, err)
	case "create_account":
		req := pb.CreateAccountRequest{}
		req.OrganizationId = organization_id
		req.AccountName = *name
		req.AccountDescription = *description
		req.AccountTypeId = int32(*type_id)
		resp, err := client.CreateAccount(mctx, &req)
		printResponse(resp, err)
	case "update_account":
		req := pb.UpdateAccountRequest{}
		req.GlAccountId = account_id
		req.Version = int32(*version)
		req.AccountName = *name
		req.AccountDescription = *description
		req.AccountTypeId = int32(*type_id)
		resp, err := client.UpdateAccount(mctx, &req)
		printResponse(resp, err)
	case "delete_account":
		req := pb.DeleteAccountRequest{}
		req.GlAccountId = account_id
		req.Version = int32(*version)
		resp, err := client.DeleteAccount(mctx, &req)
		printResponse(resp, err)
	case "get_account_by_id":
		req := pb.GetAccountByIdRequest{}
		req.GlAccountId = account_id
		resp, err := client.GetAccountById(mctx, &req)
		printResponse(resp, err)
	case "get_accounts_by_organization":
		req := pb.GetAccountsByOrganizationRequest{}
		req.OrganizationId = organization_id
		resp, err := client.GetAccountsByOrganization(mctx, &req)
		printResponse(resp, err)
	case "create_transaction":
		req := pb.CreateTransactionRequest{}
		req.OrganizationId = organization_id
		req.TransactionDate = transaction_date
		req.TransactionDescription = *description
		req.TransactionTypeId = int32(*type_id)
		if *from_party > 0 {
			req.FromPartyId = *from_party
		}
		if *to_party > 0 {
			req.ToPartyId = *to_party
		}
		req.PostedViaKey = *via_key

		if *via_date != "" {
			req.PostedViaDate = v_date
		}
		resp, err := client.CreateTransaction(mctx, &req)
		printResponse(resp, err)
	case "update_transaction":
		req := pb.UpdateTransactionRequest{}
		req.GlTransactionId = *id
		req.Version = int32(*version)
		req.TransactionDate = transaction_date
		req.TransactionDescription = *description
		req.TransactionTypeId = int32(*type_id)
		if *from_party > 0 {
			req.FromPartyId = *from_party
		}
		if *to_party > 0 {
			req.ToPartyId = *to_party
		}
		req.PostedViaKey = *via_key

		if *via_date != "" {
			req.PostedViaDate = v_date
		}
		resp, err := client.UpdateTransaction(mctx, &req)
		printResponse(resp, err)
	case "delete_transaction":
		req := pb.DeleteTransactionRequest{}
		req.GlTransactionId = *id
		req.Version = int32(*version)
		resp, err := client.DeleteTransaction(mctx, &req)
		printResponse(resp, err)
	case "get_transaction_by_id":
		req := pb.GetTransactionByIdRequest{}
		req.GlTransactionId = *id
		resp, err := client.GetTransactionById(mctx, &req)
		printResponse(resp, err)
	case "get_transaction_wrapper_by_id":
		req := pb.GetTransactionWrapperByIdRequest{}
		req.GlTransactionId = *id
		resp, err := client.GetTransactionWrapperById(mctx, &req)
		printResponse(resp, err)
	case "get_transaction_wrappers_by_date":
		req := pb.GetTransactionWrappersByDateRequest{}
		req.OrganizationId = organization_id
		req.StartDate = start_date
		req.EndDate = end_date
		resp, err := client.GetTransactionWrappersByDate(mctx, &req)
		printResponse(resp, err)
	case "add_transaction_details":
		req := pb.AddTransactionDetailsRequest{}
		req.GlTransactionId = *id
		req.GlTransactionDetails = details
		resp, err := client.AddTransactionDetails(mctx, &req)
		printResponse(resp, err)
	case "get_server_version":
		req := pb.GetServerVersionRequest{}
		req.DummyParam = 1
		resp, err := client.GetServerVersion(mctx, &req)
		printResponse(resp, err)
	}
}

// Helper to print api method response as JSON.
func printResponse(resp interface{}, err error) {
	if err == nil {
		jtext, err := json.MarshalIndent(resp, "", "  ")
		if err == nil {
			fmt.Println(string(jtext))
		}
	}
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
}

type TranDetail struct {
	Aid   string `json:"aid"`
	Amt   string `json:"amt"`
	Debit bool   `json:"debit,omitempty"`
}

func TransformDetails(tranId int64, inJson string) ([]*pb.GLTransactionDetail, error) {
	var result []*pb.GLTransactionDetail
	var err error

	var list []TranDetail

	err = json.Unmarshal([]byte(inJson), &list)
	if err != nil {
		fmt.Printf("Unmarshal err: %s\n", err)
		return nil, err
	}

	var sequence int32
	for _, detail := range list {
		fmt.Printf("Aid: %s, Amt: %s, Debit: %b\n", detail.Aid, detail.Amt, detail.Debit)
		account_id, err := dml.GuidFromString(detail.Aid)
		if err != nil {
			fmt.Printf("not a valid guid: %s\n", detail.Aid)
			return nil, InvalidParameter
		}

		if !validDecimal.MatchString(detail.Amt) {
			fmt.Printf("not a valid decimal: %s\n", detail.Amt)
			return nil, InvalidParameter
		}

		var tranDetail pb.GLTransactionDetail
		tranDetail.GlTransactionId = tranId
		sequence++
		tranDetail.SequenceNumber = sequence

		tranDetail.GlAccountId = account_id
		amount, _ := dml.DecimalFromString(detail.Amt)
		tranDetail.Amount = amount
		tranDetail.IsDebit = detail.Debit
		result = append(result, &tranDetail)
	}

	return result, err
}
