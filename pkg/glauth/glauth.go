// Copyright 2020-2021 Demian Harvill
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
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt"

	pb "github.com/gaterace/mledger/pkg/mserviceledger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"crypto/rsa"
	"io/ioutil"
)

const (
	tokenExpiredMatch   = "Token is expired"
	tokenExpiredMessage = "token is expired"
)

var NotImplemented = errors.New("not implemented")

type GlAuth struct {
	logger          log.Logger
	db              *sql.DB
	rsaPSSPublicKey *rsa.PublicKey
	glService       pb.MServiceLedgerServer
}

// Get a new projAuth instance.
func NewLedgerAuth(glService pb.MServiceLedgerServer) *GlAuth {
	svc := GlAuth{}
	svc.glService = glService
	return &svc
}

// Set the logger for the glAuth instance.
func (s *GlAuth) SetLogger(logger log.Logger) {
	s.logger = logger
}

// Set the database connection for the projAuth instance.
func (s *GlAuth) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Set the public RSA key for the projAuth instance, used to validate JWT.
func (s *GlAuth) SetPublicKey(publicKeyFile string) error {
	publicKey, err := ioutil.ReadFile(publicKeyFile)
	if err != nil {
		level.Error(s.logger).Log("what", "reading publicKeyFile", "error", err)
		return err
	}

	parsedKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		level.Error(s.logger).Log("what", "ParseRSAPublicKeyFromPEM", "error", err)
		return err
	}

	s.rsaPSSPublicKey = parsedKey
	return nil
}

// Bind our glAuth as the gRPC api server.
func (s *GlAuth) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceLedgerServer(gServer, s)

	}
	return nil
}

// Get the JWT from the gRPC request context.
func (s *GlAuth) GetJwtFromContext(ctx context.Context) (*map[string]interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get metadata from context")
	}

	tokens := md["token"]

	if (tokens == nil) || (len(tokens) == 0) {
		return nil, fmt.Errorf("cannot get token from context")
	}

	tokenString := tokens[0]

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
		level.Debug(s.logger).Log("jwt_error", err)
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid json web token")
	}

	claims := map[string]interface{}(token.Claims.(jwt.MapClaims))

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

// check to see if JWT token is expired
func (s *GlAuth) IsTokenExpired(ctx context.Context) bool {
	expired := false

	_, err := s.GetJwtFromContext(ctx)
	if err != nil {
		if err.Error() == tokenExpiredMatch {
			expired = true
		}
	}

	return expired
}

func (s *GlAuth) HasAdminAccess(ctx context.Context) (bool, int64) {
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

func (s *GlAuth) HasReadWriteAccess(ctx context.Context) (bool, int64) {
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

func (s *GlAuth) HasReadOnlyAccess(ctx context.Context) (bool, int64) {
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
func (s *GlAuth) CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.CreateOrganizationResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.CreateOrganization(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateOrganization",
		"organization", req.GetOrganizationName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing general ledger organization
func (s *GlAuth) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.UpdateOrganizationResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.UpdateOrganization(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateOrganization",
		"organization", req.GetOrganizationName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing general ledger organization
func (s *GlAuth) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.DeleteOrganizationResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.DeleteOrganization(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteOrganization",
		"organizationid", req.GetOrganizationId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger organization by id
func (s *GlAuth) GetOrganizationById(ctx context.Context, req *pb.GetOrganizationByIdRequest) (*pb.GetOrganizationByIdResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetOrganizationByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetOrganizationById(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetOrganizationById",
		"organizationid", req.GetOrganizationId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger organizations by mservice
func (s *GlAuth) GetOrganizationsByMservice(ctx context.Context, req *pb.GetOrganizationsByMserviceRequest) (*pb.GetOrganizationsByMserviceResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetOrganizationsByMserviceResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetOrganizationsByMservice(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetOrganizationsByMservice",
		"mservieid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create general ledger account type
func (s *GlAuth) CreateAccountType(ctx context.Context, req *pb.CreateAccountTypeRequest) (*pb.CreateAccountTypeResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.CreateAccountTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.CreateAccountType(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateAccountType",
		"accounttype", req.GetAccountType(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update general ledger account type
func (s *GlAuth) UpdateAccountType(ctx context.Context, req *pb.UpdateAccountTypeRequest) (*pb.UpdateAccountTypeResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.UpdateAccountTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.UpdateAccountType(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateAccountType",
		"accounttype", req.GetAccountType(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete general ledger account type
func (s *GlAuth) DeleteAccountType(ctx context.Context, req *pb.DeleteAccountTypeRequest) (*pb.DeleteAccountTypeResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.DeleteAccountTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.DeleteAccountType(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteAccountType",
		"accounttypeid", req.GetAccountTypeId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger account type by id
func (s *GlAuth) GetAccountTypeById(ctx context.Context, req *pb.GetAccountTypeByIdRequest) (*pb.GetAccountTypeByIdResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetAccountTypeByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetAccountTypeById(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetAccountTypeById",
		"accounttypeid", req.GetAccountTypeId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger account types by mservice
func (s *GlAuth) GetAccountTypesByMservice(ctx context.Context, req *pb.GetAccountTypesByMserviceRequest) (*pb.GetAccountTypesByMserviceResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetAccountTypesByMserviceResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetAccountTypesByMservice(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetAccountTypesByMservice",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create general ledger transaction type
func (s *GlAuth) CreateTransactionType(ctx context.Context, req *pb.CreateTransactionTypeRequest) (*pb.CreateTransactionTypeResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.CreateTransactionTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.CreateTransactionType(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateTransactionType",
		"transactiontype", req.GetTransactionType(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update general ledger transaction type
func (s *GlAuth) UpdateTransactionType(ctx context.Context, req *pb.UpdateTransactionTypeRequest) (*pb.UpdateTransactionTypeResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.UpdateTransactionTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.UpdateTransactionType(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateTransactionType",
		"transactiontype", req.GetTransactionType(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete general ledger transaction type
func (s *GlAuth) DeleteTransactionType(ctx context.Context, req *pb.DeleteTransactionTypeRequest) (*pb.DeleteTransactionTypeResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.DeleteTransactionTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.DeleteTransactionType(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteTransactionType",
		"transactiontypeid", req.GetTransactionTypeId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger transaction type by id
func (s *GlAuth) GetTransactionTypeById(ctx context.Context, req *pb.GetTransactionTypeByIdRequest) (*pb.GetTransactionTypeByIdResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetTransactionTypeByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetTransactionTypeById(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTransactionTypeById",
		"transactiontypeid", req.GetTransactionTypeId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger transaction types by mservice
func (s *GlAuth) GetTransactionTypesByMservice(ctx context.Context, req *pb.GetTransactionTypesByMserviceRequest) (*pb.GetTransactionTypesByMserviceResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetTransactionTypesByMserviceResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetTransactionTypesByMservice(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTransactionTypeById",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create general ledger party
func (s *GlAuth) CreateParty(ctx context.Context, req *pb.CreatePartyRequest) (*pb.CreatePartyResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.CreatePartyResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.CreateParty(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateParty",
		"party", req.GetPartyName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update general ledger party
func (s *GlAuth) UpdateParty(ctx context.Context, req *pb.UpdatePartyRequest) (*pb.UpdatePartyResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.UpdatePartyResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.UpdateParty(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateParty",
		"party", req.GetPartyName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete general ledger party
func (s *GlAuth) DeleteParty(ctx context.Context, req *pb.DeletePartyRequest) (*pb.DeletePartyResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.DeletePartyResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.DeleteParty(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteParty",
		"partyid", req.GetPartyId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger party by id
func (s *GlAuth) GetPartyById(ctx context.Context, req *pb.GetPartyByIdRequest) (*pb.GetPartyByIdResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetPartyByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetPartyById(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetPartyById",
		"partyid", req.GetPartyId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger parties by mservice
func (s *GlAuth) GetPartiesByMservice(ctx context.Context, req *pb.GetPartiesByMserviceRequest) (*pb.GetPartiesByMserviceResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetPartiesByMserviceResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetPartiesByMservice(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetPartiesByMservice",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create general ledger account
func (s *GlAuth) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.CreateAccountResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.CreateAccount(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateAccount",
		"account", req.GetAccountName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update general ledger account
func (s *GlAuth) UpdateAccount(ctx context.Context, req *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.UpdateAccountResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.UpdateAccount(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateAccount",
		"account", req.GetAccountName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete general ledger account
func (s *GlAuth) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.DeleteAccountResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasAdminAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.DeleteAccount(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteAccount",
		"accountid", req.GlAccountId,
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger account by id
func (s *GlAuth) GetAccountById(ctx context.Context, req *pb.GetAccountByIdRequest) (*pb.GetAccountByIdResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetAccountByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetAccountById(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetAccountById",
		"accountid", req.GetGlAccountId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger accounts by organization
func (s *GlAuth) GetAccountsByOrganization(ctx context.Context, req *pb.GetAccountsByOrganizationRequest) (*pb.GetAccountsByOrganizationResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetAccountsByOrganizationResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetAccountsByOrganization(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetAccountsByOrganization",
		"organizationid", req.GetOrganizationId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create general ledger transaction
func (s *GlAuth) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.CreateTransactionResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.CreateTransaction(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateTransaction",
		"organizationid", req.GetOrganizationId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update general ledger transaction
func (s *GlAuth) UpdateTransaction(ctx context.Context, req *pb.UpdateTransactionRequest) (*pb.UpdateTransactionResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.UpdateTransactionResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.UpdateTransaction(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateTransaction",
		"transactionid", req.GetGlTransactionId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete general ledger transaction
func (s *GlAuth) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteTransactionResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.DeleteTransactionResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.DeleteTransaction(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteTransaction",
		"transactionid", req.GetGlTransactionId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger transaction by id
func (s *GlAuth) GetTransactionById(ctx context.Context, req *pb.GetTransactionByIdRequest) (*pb.GetTransactionByIdResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetTransactionByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetTransactionById(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTransactionById",
		"transactionid", req.GetGlTransactionId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger transaction wrapper by id
func (s *GlAuth) GetTransactionWrapperById(ctx context.Context, req *pb.GetTransactionWrapperByIdRequest) (*pb.GetTransactionWrapperByIdResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetTransactionWrapperByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetTransactionWrapperById(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTransactionWrapperById",
		"transactionid", req.GetGlTransactionId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get general ledger transaction wrappers by date
func (s *GlAuth) GetTransactionWrappersByDate(ctx context.Context, req *pb.GetTransactionWrappersByDateRequest) (*pb.GetTransactionWrappersByDateResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.GetTransactionWrappersByDateResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadOnlyAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.GetTransactionWrappersByDate(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTransactionWrappersByDate",
		"organizationid", req.GetOrganizationId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// add transaction details
func (s *GlAuth) AddTransactionDetails(ctx context.Context, req *pb.AddTransactionDetailsRequest) (*pb.AddTransactionDetailsResponse, error) {
	start := time.Now().UnixNano()
	var err error

	resp := &pb.AddTransactionDetailsResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	ok, aid := s.HasReadWriteAccess(ctx)
	if ok {
		req.MserviceId = aid
		resp, err = s.glService.AddTransactionDetails(ctx, req)
	} else if s.IsTokenExpired(ctx) {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "AddTransactionDetails",
		"transactionid", req.GetGlTransactionId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get current server version and uptime - health check
func (s *GlAuth) GetServerVersion(ctx context.Context, req *pb.GetServerVersionRequest) (*pb.GetServerVersionResponse, error) {
	return s.glService.GetServerVersion(ctx, req)
}
