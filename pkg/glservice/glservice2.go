// Copyright 2020-2022 Demian Harvill
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

package glservice

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-kit/kit/log/level"

	"github.com/gaterace/dml-go/pkg/dml"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/mledger/pkg/mserviceledger"
)

// create general ledger transaction type
func (s *glService) CreateTransactionType(ctx context.Context, req *pb.CreateTransactionTypeRequest) (*pb.CreateTransactionTypeResponse, error) {
	resp := &pb.CreateTransactionTypeResponse{}

	sqlstring := `INSERT INTO tb_GLTransactionType (inbMserviceId, intTransactionTypeId, dtmCreated, dtmModified, 
		dtmDeleted, bitIsDeleted, intVersion, chvTransactionType) 
		VALUES (?, ?, NOW(), NOW(), NOW(), 0, 1, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetTransactionTypeId(), req.GetTransactionType())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// update general ledger transaction type
func (s *glService) UpdateTransactionType(ctx context.Context, req *pb.UpdateTransactionTypeRequest) (*pb.UpdateTransactionTypeResponse, error) {
	resp := &pb.UpdateTransactionTypeResponse{}

	sqlstring := `UPDATE tb_GLTransactionType SET dtmModified = NOW(), intVersion = ?, chvTransactionType = ?
	WHERE inbMserviceId = ? AND intTransactionTypeId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetTransactionType(), req.GetMserviceId(), req.GetTransactionTypeId(), req.GetVersion())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// delete general ledger transaction type
func (s *glService) DeleteTransactionType(ctx context.Context, req *pb.DeleteTransactionTypeRequest) (*pb.DeleteTransactionTypeResponse, error) {
	resp := &pb.DeleteTransactionTypeResponse{}

	sqlstring := `UPDATE tb_GLTransactionType SET dtmDeleted = NOW(), intVersion = ?, bitIsDeleted = 1
	WHERE inbMserviceId = ? AND intTransactionTypeId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetMserviceId(), req.GetTransactionTypeId(), req.GetVersion())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// get general ledger transaction type by id
func (s *glService) GetTransactionTypeById(ctx context.Context, req *pb.GetTransactionTypeByIdRequest) (*pb.GetTransactionTypeByIdResponse, error) {
	resp := &pb.GetTransactionTypeByIdResponse{}

	sqlstring := `SELECT inbMserviceId, intTransactionTypeId, dtmCreated, dtmModified, intVersion, chvTransactionType
	FROM tb_GLTransactionType WHERE inbMserviceId = ? AND intTransactionTypeId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created time.Time
	var modified time.Time
	var tranType pb.GLTransactionType

	err = stmt.QueryRow(req.GetMserviceId(), req.GetTransactionTypeId()).Scan(&tranType.MserviceId, &tranType.TransactionTypeId, &created, &modified, &tranType.Version, &tranType.TransactionType)

	if err == nil {
		tranType.Created = dml.DateTimeFromTime(created)
		tranType.Modified = dml.DateTimeFromTime(modified)
		resp.GlTransactionType = &tranType
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, nil
}

// get general ledger transaction types by mservice
func (s *glService) GetTransactionTypesByMservice(ctx context.Context, req *pb.GetTransactionTypesByMserviceRequest) (*pb.GetTransactionTypesByMserviceResponse, error) {
	resp := &pb.GetTransactionTypesByMserviceResponse{}

	sqlstring := `SELECT inbMserviceId, intTransactionTypeId, dtmCreated, dtmModified, intVersion, chvTransactionType
	FROM tb_GLTransactionType WHERE inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId())
	if err != nil {
		level.Error(s.logger).Log("what", "Query", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created time.Time
		var modified time.Time
		var tranType pb.GLTransactionType
		err := rows.Scan(&tranType.MserviceId, &tranType.TransactionTypeId, &created, &modified, &tranType.Version, &tranType.TransactionType)
		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		tranType.Created = dml.DateTimeFromTime(created)
		tranType.Modified = dml.DateTimeFromTime(modified)

		resp.GlTransactionTypes = append(resp.GlTransactionTypes, &tranType)
	}

	return resp, nil
}

// create general ledger party
func (s *glService) CreateParty(ctx context.Context, req *pb.CreatePartyRequest) (*pb.CreatePartyResponse, error) {
	resp := &pb.CreatePartyResponse{}

	sqlstring := `INSERT INTO tb_GLParty (inbMserviceId, inbPartyId, dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, 
		chvPartyName) VALUES(?, ?, NOW(), NOW(), NOW(), 0, 1, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetPartyId(), req.GetPartyName())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// update general ledger party
func (s *glService) UpdateParty(ctx context.Context, req *pb.UpdatePartyRequest) (*pb.UpdatePartyResponse, error) {
	resp := &pb.UpdatePartyResponse{}

	sqlstring := `UPDATE tb_GLParty SET dtmModified = NOW(), intVersion = ?, chvPartyName = ? WHERE 
	inbMserviceId = ? AND inbPartyId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetPartyName(), req.GetMserviceId(), req.GetPartyId(), req.GetVersion())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// delete general ledger party
func (s *glService) DeleteParty(ctx context.Context, req *pb.DeletePartyRequest) (*pb.DeletePartyResponse, error) {
	resp := &pb.DeletePartyResponse{}

	sqlstring := `UPDATE tb_GLParty SET dtmDeleted = NOW(), intVersion = ?, bitIsDeleted = 1 WHERE 
	inbMserviceId = ? AND inbPartyId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetMserviceId(), req.GetPartyId(), req.GetVersion())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// get general ledger party by id
func (s *glService) GetPartyById(ctx context.Context, req *pb.GetPartyByIdRequest) (*pb.GetPartyByIdResponse, error) {
	resp := &pb.GetPartyByIdResponse{}

	sqlstring := `SELECT inbMserviceId, inbPartyId, dtmCreated, dtmModified, intVersion, chvPartyName
	FROM  tb_GLParty WHERE inbMserviceId = ? AND inbPartyId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created time.Time
	var modified time.Time
	var party pb.GLParty

	err = stmt.QueryRow(req.GetMserviceId(), req.GetPartyId()).Scan(&party.MserviceId, &party.PartyId, &created, &modified,
		&party.Version, &party.PartyName)

	if err == nil {
		party.Created = dml.DateTimeFromTime(created)
		party.Modified = dml.DateTimeFromTime(modified)
		resp.GlParty = &party
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, nil
}

// get general ledger parties by mservice
func (s *glService) GetPartiesByMservice(ctx context.Context, req *pb.GetPartiesByMserviceRequest) (*pb.GetPartiesByMserviceResponse, error) {
	resp := &pb.GetPartiesByMserviceResponse{}

	sqlstring := `SELECT inbMserviceId, inbPartyId, dtmCreated, dtmModified, intVersion, chvPartyName
	FROM  tb_GLParty WHERE inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId())
	if err != nil {
		level.Error(s.logger).Log("what", "Query", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created time.Time
		var modified time.Time
		var party pb.GLParty

		err := rows.Scan(&party.MserviceId, &party.PartyId, &created, &modified, &party.Version, &party.PartyName)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		party.Created = dml.DateTimeFromTime(created)
		party.Modified = dml.DateTimeFromTime(modified)
		resp.GlParties = append(resp.GlParties, &party)
	}

	return resp, nil
}

// create general ledger account
func (s *glService) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	resp := &pb.CreateAccountResponse{}

	sqlstring := `INSERT INTO tb_GLAccount (uidGlAccountId, dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, 
		inbMserviceId, uidOrganizationId, chvAccountName, chvAccountDescription, intAccountTypeId) 
		VALUES (?, NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	glId := dml.NewGuid()

	guid := req.GetOrganizationId()

	res, err := stmt.Exec(glId.Guid, req.GetMserviceId(), guid.Guid, req.GetAccountName(), req.GetAccountDescription(), req.GetAccountTypeId())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = 1
			resp.GlAccountId = glId
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// update general ledger account
func (s *glService) UpdateAccount(ctx context.Context, req *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	resp := &pb.UpdateAccountResponse{}

	sqlstring := `UPDATE tb_GLAccount SET dtmModified = NOW(), intVersion = ?, chvAccountName = ?, chvAccountDescription = ?, 
	intAccountTypeId = ? WHERE inbMserviceId = ? AND uidGlAccountId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetAccountName(), req.GetAccountDescription(), req.GetAccountTypeId(),
		req.GetMserviceId(), req.GetGlAccountId().Guid, req.GetVersion())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// delete general ledger account
func (s *glService) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	resp := &pb.DeleteAccountResponse{}

	sqlstring := `UPDATE tb_GLAccount SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = ? 
	WHERE inbMserviceId = ? AND uidGlAccountId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetMserviceId(), req.GetGlAccountId().Guid, req.GetVersion())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, nil
}

// get general ledger account by id
func (s *glService) GetAccountById(ctx context.Context, req *pb.GetAccountByIdRequest) (*pb.GetAccountByIdResponse, error) {
	resp := &pb.GetAccountByIdResponse{}

	sqlstring := `SELECT a.uidGlAccountId, a.dtmCreated, a.dtmModified, a.intVersion, 
	a.inbMserviceId, a.uidOrganizationId, a.chvAccountName, a.chvAccountDescription, a.intAccountTypeId,
	o.chvOrganizationName, t.chvAccountType
	FROM tb_GLAccount AS a
	JOIN tb_GLOrganization AS o
	ON a.uidOrganizationId = o.uidOrganizationId
	JOIN tb_GLAccountType AS t
    ON a.inbMserviceId = t.inbMserviceId AND a.intAccountTypeId = t.intAccountTypeId
	WHERE a.inbMserviceId = ? AND a.uidGlAccountId = ? AND a.bitIsDeleted = 0 AND o.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var acctGid []byte
	var orgGid []byte
	var created time.Time
	var modified time.Time
	var acct pb.GLAccount

	err = stmt.QueryRow(req.GetMserviceId(), req.GlAccountId.Guid).Scan(&acctGid, &created, &modified, &acct.Version,
		&acct.MserviceId, &orgGid, &acct.AccountName, &acct.AccountDescription, &acct.AccountTypeId,
		&acct.OrganizationName, &acct.AccountType)

	if err == nil {
		acct.GlAccountId, _ = dml.GuidFromBytes(acctGid)
		acct.OrganizationId, _ = dml.GuidFromBytes(orgGid)
		acct.Created = dml.DateTimeFromTime(created)
		acct.Modified = dml.DateTimeFromTime(modified)
		resp.GlAccount = &acct
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, nil
}

// get general ledger accounts by organization
func (s *glService) GetAccountsByOrganization(ctx context.Context, req *pb.GetAccountsByOrganizationRequest) (*pb.GetAccountsByOrganizationResponse, error) {
	resp := &pb.GetAccountsByOrganizationResponse{}

	sqlstring := `SELECT a.uidGlAccountId, a.dtmCreated, a.dtmModified, a.intVersion, 
	a.inbMserviceId, a.uidOrganizationId, a.chvAccountName, a.chvAccountDescription, a.intAccountTypeId,
	o.chvOrganizationName, t.chvAccountType
	FROM tb_GLAccount AS a
	JOIN tb_GLOrganization AS o
	ON a.uidOrganizationId = o.uidOrganizationId AND a.inbMserviceId = o.inbMserviceId
	JOIN tb_GLAccountType AS t
    ON a.inbMserviceId = t.inbMserviceId AND a.intAccountTypeId = t.intAccountTypeId
	WHERE a.inbMserviceId = ? AND a.uidOrganizationId = ? AND a.bitIsDeleted = 0 AND o.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId(), req.GetOrganizationId().Guid)
	if err != nil {
		level.Error(s.logger).Log("what", "Query", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var acctGid []byte
		var orgGid []byte
		var created time.Time
		var modified time.Time
		var acct pb.GLAccount

		err := rows.Scan(&acctGid, &created, &modified, &acct.Version,
			&acct.MserviceId, &orgGid, &acct.AccountName, &acct.AccountDescription, &acct.AccountTypeId,
			&acct.OrganizationName, &acct.AccountType)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		acct.GlAccountId, _ = dml.GuidFromBytes(acctGid)
		var oid dml.Guid
		oid.Guid = orgGid
		acct.OrganizationId = &oid
		acct.Created = dml.DateTimeFromTime(created)
		acct.Modified = dml.DateTimeFromTime(modified)
		resp.GlAccounts = append(resp.GlAccounts, &acct)
	}

	return resp, nil
}
