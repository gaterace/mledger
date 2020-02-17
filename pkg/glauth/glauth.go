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

// Package glauth provides authorization for each GRPC method in MServiceLedger.
// The JWT extracted from the GRPC request context is used for each delegating method.
package glauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/mledger/pkg/mserviceledger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"crypto/rsa"
	"io/ioutil"
)

var NotImplemented = errors.New("not implemented")

type glAuth struct {
	logger          *log.Logger
	db              *sql.DB
	rsaPSSPublicKey *rsa.PublicKey
	glService       pb.MServiceLedgerServer
}

// Get a new projAuth instance.
func NewLedgerAuth(glService pb.MServiceLedgerServer) *glAuth {
	svc := glAuth{}
	svc.glService = glService
	return &svc
}

// Set the logger for the glAuth instance.
func (s *glAuth) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// Set the database connection for the projAuth instance.
func (s *glAuth) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Set the public RSA key for the projAuth instance, used to validate JWT.
func (s *glAuth) SetPublicKey(publicKeyFile string) error {
	publicKey, err := ioutil.ReadFile(publicKeyFile)
	if err != nil {
		s.logger.Printf("error reading publicKeyFile: %v\n", err)
		return err
	}

	parsedKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		s.logger.Printf("error parsing publicKeyFile: %v\n", err)
		return err
	}

	s.rsaPSSPublicKey = parsedKey
	return nil
}

// Bind our glAuth as the gRPC api server.
func (s *glAuth) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceLedgerServer(gServer, s)

	}
	return nil
}

// Get the JWT from the gRPC request context.
func (s *glAuth) GetJwtFromContext(ctx context.Context) (*map[string]interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get metadata from context")
	}

	tokens := md["token"]

	if (tokens == nil) || (len(tokens) == 0) {
		return nil, fmt.Errorf("cannot get token from context")
	}

	tokenString := tokens[0]

	s.logger.Printf("tokenString: %s\n", tokenString)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		method := token.Method.Alg()
		if method != "PS256" {

			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// return []byte(mySigningKey), nil
		return s.rsaPSSPublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid json web token")
	}

	claims := map[string]interface{}(token.Claims.(jwt.MapClaims))

	s.logger.Printf("claims: %v\n", claims)

	return &claims, nil

}

// Get the clain value as an int64.
func GetInt64FromClaims(claims *map[string]interface{}, key string) int64 {
	var val int64

	if claims != nil {
		cval := (*claims)[key]
		if fval, ok := cval.(float64); ok {
			val = int64(fval)
		}
	}

	return val
}

// Get the claim value as a string.
func GetStringFromClaims(claims *map[string]interface{}, key string) string {
	var val string

	if claims != nil {
		cval := (*claims)[key]
		if sval, ok := cval.(string); ok {
			val = sval
		}
	}

	return val
}

func (s *glAuth) HasAdminAccess(ctx context.Context) (bool, int64) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		ledger := GetStringFromClaims(claims, "ledger")
		if ledger == "gladmin" {
			aid := GetInt64FromClaims(claims, "aid")
			return true, aid
		}
	}

	return false, 0
}

func (s *glAuth) HasReadWriteAccess(ctx context.Context) (bool, int64) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		ledger := GetStringFromClaims(claims, "ledger")
		if (ledger == "gladmin") || (ledger == "glrw") {
			aid := GetInt64FromClaims(claims, "aid")
			return true, aid
		}
	}

	return false, 0
}

func (s *glAuth) HasReadOnlyAccess(ctx context.Context) (bool, int64) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		ledger := GetStringFromClaims(claims, "ledger")
		if (ledger == "gladmin") || (ledger == "glrw") || (ledger == "glro") {
			aid := GetInt64FromClaims(claims, "aid")
			return true, aid
		}
	}

	return false, 0
}

// create a new general ledger organization
func (s *glAuth) CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.CreateOrganization(ctx, req)
	}

	resp := &pb.CreateOrganizationResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// update an existing general ledger organization
func (s *glAuth) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.UpdateOrganization(ctx, req)
	}

	resp := &pb.UpdateOrganizationResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// delete an existing general ledger organization
func (s *glAuth) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.DeleteOrganization(ctx, req)
	}

	resp := &pb.DeleteOrganizationResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger organization by id
func (s *glAuth) GetOrganizationById(ctx context.Context, req *pb.GetOrganizationByIdRequest) (*pb.GetOrganizationByIdResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetOrganizationById(ctx, req)
	}

	resp := &pb.GetOrganizationByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger organizations by mservice
func (s *glAuth) GetOrganizationsByMservice(ctx context.Context, req *pb.GetOrganizationsByMserviceRequest) (*pb.GetOrganizationsByMserviceResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetOrganizationsByMservice(ctx, req)
	}

	resp := &pb.GetOrganizationsByMserviceResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// create general ledger account type
func (s *glAuth) CreateAccountType(ctx context.Context, req *pb.CreateAccountTypeRequest) (*pb.CreateAccountTypeResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.CreateAccountType(ctx, req)
	}

	resp := &pb.CreateAccountTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// update general ledger account type
func (s *glAuth) UpdateAccountType(ctx context.Context, req *pb.UpdateAccountTypeRequest) (*pb.UpdateAccountTypeResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.UpdateAccountType(ctx, req)
	}

	resp := &pb.UpdateAccountTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// delete general ledger account type
func (s *glAuth) DeleteAccountType(ctx context.Context, req *pb.DeleteAccountTypeRequest) (*pb.DeleteAccountTypeResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.DeleteAccountType(ctx, req)
	}

	resp := &pb.DeleteAccountTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger account type by id
func (s *glAuth) GetAccountTypeById(ctx context.Context, req *pb.GetAccountTypeByIdRequest) (*pb.GetAccountTypeByIdResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetAccountTypeById(ctx, req)
	}

	resp := &pb.GetAccountTypeByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger account types by mservice
func (s *glAuth) GetAccountTypesByMservice(ctx context.Context, req *pb.GetAccountTypesByMserviceRequest) (*pb.GetAccountTypesByMserviceResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetAccountTypesByMservice(ctx, req)
	}

	resp := &pb.GetAccountTypesByMserviceResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// create general ledger transaction type
func (s *glAuth) CreateTransactionType(ctx context.Context, req *pb.CreateTransactionTypeRequest) (*pb.CreateTransactionTypeResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.CreateTransactionType(ctx, req)
	}

	resp := &pb.CreateTransactionTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// update general ledger transaction type
func (s *glAuth) UpdateTransactionType(ctx context.Context, req *pb.UpdateTransactionTypeRequest) (*pb.UpdateTransactionTypeResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.UpdateTransactionType(ctx, req)
	}

	resp := &pb.UpdateTransactionTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// delete general ledger transaction type
func (s *glAuth) DeleteTransactionType(ctx context.Context, req *pb.DeleteTransactionTypeRequest) (*pb.DeleteTransactionTypeResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.DeleteTransactionType(ctx, req)
	}

	resp := &pb.DeleteTransactionTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger transaction type by id
func (s *glAuth) GetTransactionTypeById(ctx context.Context, req *pb.GetTransactionTypeByIdRequest) (*pb.GetTransactionTypeByIdResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetTransactionTypeById(ctx, req)
	}

	resp := &pb.GetTransactionTypeByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger transaction types by mservice
func (s *glAuth) GetTransactionTypesByMservice(ctx context.Context, req *pb.GetTransactionTypesByMserviceRequest) (*pb.GetTransactionTypesByMserviceResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetTransactionTypesByMservice(ctx, req)
	}

	resp := &pb.GetTransactionTypesByMserviceResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// create general ledger party
func (s *glAuth) CreateParty(ctx context.Context, req *pb.CreatePartyRequest) (*pb.CreatePartyResponse, error) {
	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.CreateParty(ctx, req)
	}

	resp := &pb.CreatePartyResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// update general ledger party
func (s *glAuth) UpdateParty(ctx context.Context, req *pb.UpdatePartyRequest) (*pb.UpdatePartyResponse, error) {
	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.UpdateParty(ctx, req)
	}

	resp := &pb.UpdatePartyResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// delete general ledger party
func (s *glAuth) DeleteParty(ctx context.Context, req *pb.DeletePartyRequest) (*pb.DeletePartyResponse, error) {
	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.DeleteParty(ctx, req)
	}

	resp := &pb.DeletePartyResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger party by id
func (s *glAuth) GetPartyById(ctx context.Context, req *pb.GetPartyByIdRequest) (*pb.GetPartyByIdResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetPartyById(ctx, req)
	}

	resp := &pb.GetPartyByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger parties by mservice
func (s *glAuth) GetPartiesByMservice(ctx context.Context, req *pb.GetPartiesByMserviceRequest) (*pb.GetPartiesByMserviceResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetPartiesByMservice(ctx, req)
	}

	resp := &pb.GetPartiesByMserviceResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// create general ledger account
func (s *glAuth) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.CreateAccount(ctx, req)
	}

	resp := &pb.CreateAccountResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// update general ledger account
func (s *glAuth) UpdateAccount(ctx context.Context, req *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.UpdateAccount(ctx, req)
	}

	resp := &pb.UpdateAccountResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// delete general ledger account
func (s *glAuth) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.DeleteAccount(ctx, req)
	}

	resp := &pb.DeleteAccountResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger account by id
func (s *glAuth) GetAccountById(ctx context.Context, req *pb.GetAccountByIdRequest) (*pb.GetAccountByIdResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetAccountById(ctx, req)
	}

	resp := &pb.GetAccountByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger accounts by organization
func (s *glAuth) GetAccountsByOrganization(ctx context.Context, req *pb.GetAccountsByOrganizationRequest) (*pb.GetAccountsByOrganizationResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetAccountsByOrganization(ctx, req)
	}

	resp := &pb.GetAccountsByOrganizationResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// create general ledger transaction
func (s *glAuth) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.CreateTransaction(ctx, req)
	}

	resp := &pb.CreateTransactionResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// update general ledger transaction
func (s *glAuth) UpdateTransaction(ctx context.Context, req *pb.UpdateTransactionRequest) (*pb.UpdateTransactionResponse, error) {
	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.UpdateTransaction(ctx, req)
	}

	resp := &pb.UpdateTransactionResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// delete general ledger transaction
func (s *glAuth) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteTransactionResponse, error) {
	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.DeleteTransaction(ctx, req)
	}

	resp := &pb.DeleteTransactionResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger transaction by id
func (s *glAuth) GetTransactionById(ctx context.Context, req *pb.GetTransactionByIdRequest) (*pb.GetTransactionByIdResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetTransactionById(ctx, req)
	}

	resp := &pb.GetTransactionByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger transaction wrapper by id
func (s *glAuth) GetTransactionWrapperById(ctx context.Context, req *pb.GetTransactionWrapperByIdRequest) (*pb.GetTransactionWrapperByIdResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetTransactionWrapperById(ctx, req)
	}

	resp := &pb.GetTransactionWrapperByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// get general ledger transaction wrappers by date
func (s *glAuth) GetTransactionWrappersByDate(ctx context.Context, req *pb.GetTransactionWrappersByDateRequest) (*pb.GetTransactionWrappersByDateResponse, error) {
	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.GetTransactionWrappersByDate(ctx, req)
	}

	resp := &pb.GetTransactionWrappersByDateResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}

// add transaction details
func (s *glAuth) AddTransactionDetails(ctx context.Context, req *pb.AddTransactionDetailsRequest) (*pb.AddTransactionDetailsResponse, error) {
	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		return s.glService.AddTransactionDetails(ctx, req)
	}

	resp := &pb.AddTransactionDetailsResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"
	return resp, nil
}
