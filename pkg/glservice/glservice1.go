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

// Package  projservice provides the implemantation for the MServiceLedger gRPC service.
package glservice

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gaterace/dml-go/pkg/dml"
	pb "github.com/gaterace/mledger/pkg/mserviceledger"
	"google.golang.org/grpc"
)

var NotImplemented = errors.New("not implemented")

var nameValidator = regexp.MustCompile("^[a-z0-9_\\-]{1,32}$")

var emptyDateString = "0000-00-00 00:00:00"

type glService struct {
	logger *log.Logger
	db     *sql.DB
	startSecs int64
}

// Get a new projService instance.
func NewGlService() *glService {
	svc := glService{}
	svc.startSecs = time.Now().Unix()
	return &svc
}

// Set the logger for the glService instance.
func (s *glService) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// Set the database connection for the glService instance.
func (s *glService) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Bind this glService the gRPC server api.
func (s *glService) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceLedgerServer(gServer, s)

	}
	return nil
}

// create a new general ledger organization
func (s *glService) CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	s.logger.Printf("CreateOrganization called, name: %s\n", req.GetOrganizationName())
	resp := &pb.CreateOrganizationResponse{}
	if !nameValidator.MatchString(req.GetOrganizationName()) {
		resp.ErrorCode = 510
		resp.ErrorMessage = "organization_name invalid format"
		return resp, nil
	}

	sqlstring := `INSERT INTO tb_GLOrganization 
	(uidOrganizationId, dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId,
		chvOrganizationName, dtmFromDate, dtmToDate) VALUES(?, NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var from_date time.Time

	var to_date sql.NullTime

	glId := dml.NewGuid()

	from_date = req.GetFromDate().TimeFromDateTime()
	if req.GetToDate() != nil {

		to_date.Time = req.GetToDate().TimeFromDateTime()
		to_date.Valid = true
	}

	res, err := stmt.Exec(glId.GetGuid(), req.GetMserviceId(), req.GetOrganizationName(), from_date, to_date)
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = 1
			resp.OrganizationId = glId
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// update an existing general ledger organization
func (s *glService) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	s.logger.Printf("UpdateOrganization called, name: %s, guid: %v\n", req.GetOrganizationName(), req.GetOrganizationId())
	resp := &pb.UpdateOrganizationResponse{}
	if !nameValidator.MatchString(req.GetOrganizationName()) {
		resp.ErrorCode = 510
		resp.ErrorMessage = "organization_name invalid format"
		return resp, nil
	}

	sqlstring := `UPDATE tb_GLOrganization SET dtmModified = NOW(), intVersion = ?, chvOrganizationName = ?, dtmFromDate = ?, dtmToDate =  ? WHERE uidOrganizationId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var from_date time.Time
	var to_date sql.NullTime

	from_date = req.GetFromDate().TimeFromDateTime()
	if req.GetToDate() != nil {
		to_date.Time = req.GetToDate().TimeFromDateTime()
		to_date.Valid = true
	}

	guid := req.GetOrganizationId()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetOrganizationName(), from_date, to_date, guid.Guid, req.GetMserviceId(), req.GetVersion())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// delete an existing general ledger organization
func (s *glService) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	s.logger.Printf("DeleteOrganization called, guid: %v\n", req.GetOrganizationId())
	resp := &pb.DeleteOrganizationResponse{}

	sqlstring := `UPDATE tb_GLOrganization SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = ? WHERE uidOrganizationId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	guid := req.GetOrganizationId()

	res, err := stmt.Exec(req.GetVersion()+1, guid.Guid, req.GetMserviceId(), req.GetVersion())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}
	return resp, nil
}

// get general ledger organization by id
func (s *glService) GetOrganizationById(ctx context.Context, req *pb.GetOrganizationByIdRequest) (*pb.GetOrganizationByIdResponse, error) {
	s.logger.Printf("GetOrganizationById called, guid: %v\n", req.GetOrganizationId())
	resp := &pb.GetOrganizationByIdResponse{}

	sqlstring := `SELECT uidOrganizationId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvOrganizationName, dtmFromDate, dtmToDate FROM tb_GLOrganization WHERE 
	uidOrganizationId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	guid := req.GetOrganizationId()

	var gid []byte
	var created time.Time
	var modified time.Time
	var start_date time.Time
	var end_date sql.NullTime
	var org pb.GLOrganization

	err = stmt.QueryRow(guid.Guid, req.GetMserviceId()).Scan(&gid, &created, &modified, &org.Version, &org.MserviceId, &org.OrganizationName, &start_date, &end_date)
	if err == nil {
		org.OrganizationId, _ = dml.GuidFromBytes(gid)
		org.Created = dml.DateTimeFromTime(created)
		org.Modified = dml.DateTimeFromTime(modified)
		org.FromDate = dml.DateTimeFromTime(start_date)
		if end_date.Valid {
			org.ToDate = dml.DateTimeFromTime(end_date.Time)
			s.logger.Printf("end_date: %s, millis: %d\n", end_date, org.ToDate.Milliseconds)
		}

		resp.GlOrganization = &org
		resp.ErrorCode = 0
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		s.logger.Printf("queryRow failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, nil
}

// get general ledger organizations by mservice
func (s *glService) GetOrganizationsByMservice(ctx context.Context, req *pb.GetOrganizationsByMserviceRequest) (*pb.GetOrganizationsByMserviceResponse, error) {
	s.logger.Printf("GetOrganizationsByMservice called, id: %d\n", req.GetMserviceId())
	resp := &pb.GetOrganizationsByMserviceResponse{}

	sqlstring := `SELECT uidOrganizationId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvOrganizationName, dtmFromDate, dtmToDate FROM tb_GLOrganization WHERE 
	inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId())
	if err != nil {
		s.logger.Printf("query failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var gid []byte
		var created time.Time
		var modified time.Time
		var start_date time.Time
		var end_date sql.NullTime

		var org pb.GLOrganization

		err := rows.Scan(&gid, &created, &modified, &org.Version, &org.MserviceId, &org.OrganizationName, &start_date, &end_date)

		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		org.OrganizationId, _ = dml.GuidFromBytes(gid)
		org.Created = dml.DateTimeFromTime(created)
		org.Modified = dml.DateTimeFromTime(modified)
		org.FromDate = dml.DateTimeFromTime(start_date)
		if end_date.Valid {
			org.ToDate = dml.DateTimeFromTime(end_date.Time)
		}

		resp.GlOrganizations = append(resp.GlOrganizations, &org)
	}

	return resp, nil
}

// create general ledger account type
func (s *glService) CreateAccountType(ctx context.Context, req *pb.CreateAccountTypeRequest) (*pb.CreateAccountTypeResponse, error) {
	s.logger.Printf("CreateAccountType called, id: %d\n", req.GetAccountTypeId())
	resp := &pb.CreateAccountTypeResponse{}

	sqlstring := `INSERT INTO tb_GLAccountType (inbMserviceId, intAccountTypeId, dtmCreated, 
		dtmModified, dtmDeleted, bitIsDeleted, intVersion, chvAccountType) 
		VALUES (?, ?, NOW(), NOW(), NOW(), 0, 1,?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetAccountTypeId(), req.GetAccountType())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// update general ledger account type
func (s *glService) UpdateAccountType(ctx context.Context, req *pb.UpdateAccountTypeRequest) (*pb.UpdateAccountTypeResponse, error) {
	s.logger.Printf("UpdateAccountType called, id: %d\n", req.GetAccountTypeId())
	resp := &pb.UpdateAccountTypeResponse{}

	sqlstring := `UPDATE tb_GLAccountType SET dtmModified = NOW(), intVersion = ?, chvAccountType = ? 
	WHERE inbMserviceId = ? AND intAccountTypeId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetAccountType(), req.GetMserviceId(), req.GetAccountTypeId(), req.GetVersion())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// delete general ledger account type
func (s *glService) DeleteAccountType(ctx context.Context, req *pb.DeleteAccountTypeRequest) (*pb.DeleteAccountTypeResponse, error) {
	s.logger.Printf("DeleteAccountType called, id: %d\n", req.GetAccountTypeId())
	resp := &pb.DeleteAccountTypeResponse{}

	sqlstring := `UPDATE tb_GLAccountType SET dtmDeleted = NOW(), bitIsDeleted = 1,  intVersion = ? 
	WHERE inbMserviceId = ? AND intAccountTypeId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetMserviceId(), req.GetAccountTypeId(), req.GetVersion())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// get general ledger account type by id
func (s *glService) GetAccountTypeById(ctx context.Context, req *pb.GetAccountTypeByIdRequest) (*pb.GetAccountTypeByIdResponse, error) {
	s.logger.Printf("GetAccountTypeById called, id: %d\n", req.GetAccountTypeId())
	resp := &pb.GetAccountTypeByIdResponse{}

	sqlstring := `SELECT inbMserviceId, intAccountTypeId, dtmCreated, dtmModified, intVersion, chvAccountType
	FROM tb_GLAccountType WHERE inbMserviceId = ? AND intAccountTypeId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created time.Time
	var modified time.Time
	var acctType pb.GLAccountType

	err = stmt.QueryRow(req.GetMserviceId(), req.GetAccountTypeId()).Scan(&acctType.MserviceId, &acctType.AccountTypeId, &created,
		&modified, &acctType.Version, &acctType.AccountType)
	if err == nil {
		acctType.Created = dml.DateTimeFromTime(created)
		acctType.Modified = dml.DateTimeFromTime(modified)
		resp.GlAccountType = &acctType
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		s.logger.Printf("queryRow failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, nil
}

// get general ledger account types by mservice
func (s *glService) GetAccountTypesByMservice(ctx context.Context, req *pb.GetAccountTypesByMserviceRequest) (*pb.GetAccountTypesByMserviceResponse, error) {
	s.logger.Printf("GetAccountTypesByMservice called, mservice: %d\n", req.GetMserviceId())
	resp := &pb.GetAccountTypesByMserviceResponse{}

	sqlstring := `SELECT inbMserviceId, intAccountTypeId, dtmCreated, dtmModified, intVersion, chvAccountType
	FROM tb_GLAccountType WHERE inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId())
	if err != nil {
		s.logger.Printf("query failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created time.Time
		var modified time.Time
		var acctType pb.GLAccountType
		err := rows.Scan(&acctType.MserviceId, &acctType.AccountTypeId, &created,
			&modified, &acctType.Version, &acctType.AccountType)
		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		acctType.Created = dml.DateTimeFromTime(created)
		acctType.Modified = dml.DateTimeFromTime(modified)

		resp.GlAccountTypes = append(resp.GlAccountTypes, &acctType)
	}

	return resp, nil
}
