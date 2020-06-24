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

package glservice

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-kit/kit/log/level"

	"github.com/gaterace/dml-go/pkg/dml"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/mledger/pkg/mserviceledger"

	sdec "github.com/shopspring/decimal"
)

// Generic response to set specific API method response.
type genericResponse struct {
	ErrorCode    int32
	ErrorMessage string
}

// create general ledger transaction
func (s *glService) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	resp := &pb.CreateTransactionResponse{}

	sqlstring := `INSERT INTO tb_GLTransaction (dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId,
	uidOrganizationId, dtmTransactionDate, chvTransactionDescription, intTransactionTypeId, inbFromPartyId, inbToPartyId,
	chvPostedViaKey, dtmPostedViaDate) VALUES (NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var from_party sql.NullInt64
	var to_party sql.NullInt64
	var via_key sql.NullString
	var via_date sql.NullTime

	if req.GetFromPartyId() != 0 {
		from_party.Int64 = req.GetFromPartyId()
		from_party.Valid = true
	}

	if req.GetToPartyId() != 0 {
		to_party.Int64 = req.GetToPartyId()
		to_party.Valid = true
	}

	if req.GetPostedViaKey() != "" {
		via_key.String = req.GetPostedViaKey()
		via_key.Valid = true
	}

	if req.GetPostedViaDate() != nil {
		via_date.Time = req.GetPostedViaDate().TimeFromDateTime()
		via_date.Valid = true
	}

	res, err := stmt.Exec(req.GetMserviceId(), req.GetOrganizationId().Guid, req.GetTransactionDate().TimeFromDateTime(), req.GetTransactionDescription(), req.GetTransactionTypeId(), &from_party, &to_party, &via_key, &via_date)

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			transactionId, err := res.LastInsertId()
			if err != nil {
				level.Error(s.logger).Log("what", "LastInsertId", "error", err)
			} else {
				level.Debug(s.logger).Log("transactionId", transactionId)
				resp.GlTransactionId = transactionId
				resp.Version = 1
			}
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

// update general ledger transaction
func (s *glService) UpdateTransaction(ctx context.Context, req *pb.UpdateTransactionRequest) (*pb.UpdateTransactionResponse, error) {
	resp := &pb.UpdateTransactionResponse{}

	sqlstring := `UPDATE tb_GLTransaction SET dtmModified = NOW(), intVersion = ?, dtmTransactionDate = ?, chvTransactionDescription= ?,
	intTransactionTypeId = ?, inbFromPartyId = ?, inbToPartyId = ?, chvPostedViaKey = ?, dtmPostedViaDate = ?
	WHERE  inbGlTransactionId = ? AND intVersion = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var from_party sql.NullInt64
	var to_party sql.NullInt64
	var via_key sql.NullString
	var via_date sql.NullTime

	if req.GetFromPartyId() != 0 {
		from_party.Int64 = req.GetFromPartyId()
		from_party.Valid = true
	}

	if req.GetToPartyId() != 0 {
		to_party.Int64 = req.GetToPartyId()
		to_party.Valid = true
	}

	if req.GetPostedViaKey() != "" {
		via_key.String = req.GetPostedViaKey()
		via_key.Valid = true
	}

	if req.GetPostedViaDate() != nil {
		via_date.Time = req.GetPostedViaDate().TimeFromDateTime()
		via_date.Valid = true
	}

	res, err := stmt.Exec(req.GetVersion()+1, req.GetTransactionDate().TimeFromDateTime(), req.GetTransactionDescription(), req.GetTransactionTypeId(), &from_party,
		&to_party, &via_key, &via_date, req.GetGlTransactionId(), req.GetVersion(), req.GetMserviceId())

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

// delete general ledger transaction
func (s *glService) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteTransactionResponse, error) {
	resp := &pb.DeleteTransactionResponse{}

	sqlstring := `UPDATE tb_GLTransaction SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = ? 
	WHERE inbGlTransactionId = ? AND  intVersion = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetGlTransactionId(), req.GetVersion(), req.GetMserviceId())

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

// get general ledger transaction by id
func (s *glService) GetTransactionById(ctx context.Context, req *pb.GetTransactionByIdRequest) (*pb.GetTransactionByIdResponse, error) {
	resp := &pb.GetTransactionByIdResponse{}

	gResp, tran := s.GetTransactionHelper(req.GetGlTransactionId(), req.GetMserviceId())
	if gResp.ErrorCode == 0 {
		resp.GlTransaction = tran
	} else {
		resp.ErrorCode = gResp.ErrorCode
		resp.ErrorMessage = gResp.ErrorMessage
	}

	return resp, nil
}

// get general ledger transaction wrapper by id
func (s *glService) GetTransactionWrapperById(ctx context.Context, req *pb.GetTransactionWrapperByIdRequest) (*pb.GetTransactionWrapperByIdResponse, error) {
	resp := &pb.GetTransactionWrapperByIdResponse{}

	gResp, tran := s.GetTransactionHelper(req.GetGlTransactionId(), req.GetMserviceId())
	if gResp.ErrorCode != 0 {
		resp.ErrorCode = gResp.ErrorCode
		resp.ErrorMessage = gResp.ErrorMessage
	}

	wrap := ConvertTransactionToWrapper(tran)

	sqlstring := `SELECT d.inbGlTransactionId, d.intSequenceNumber, d.uidGlAccountId, d.decAmount, d.bitIsDebit
	FROM tb_GLTransactionDetail AS d
	WHERE d.inbGlTransactionId = ?
	ORDER BY d.intSequenceNumber`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetGlTransactionId())

	if err != nil {
		level.Error(s.logger).Log("what", "Query", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()

	for rows.Next() {
		var gid []byte
		var amount string
		var detail pb.GLTransactionDetail

		err := rows.Scan(&detail.GlTransactionId, &detail.SequenceNumber, &gid, &amount, &detail.IsDebit)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		detail.GlAccountId, _ = dml.GuidFromBytes(gid)
		amt, err := dml.DecimalFromString(amount)
		if err == nil {
			detail.Amount = amt
		}

		wrap.GlTransactionDetails = append(wrap.GlTransactionDetails, &detail)

	}

	resp.GlTransactionWrapper = wrap

	return resp, nil
}

// get general ledger transaction wrappers by date
func (s *glService) GetTransactionWrappersByDate(ctx context.Context, req *pb.GetTransactionWrappersByDateRequest) (*pb.GetTransactionWrappersByDateResponse, error) {
	resp := &pb.GetTransactionWrappersByDateResponse{}

	sqlstring := `SELECT t.inbGlTransactionId, t.dtmCreated, t.dtmModified, t.intVersion, t.inbMserviceId, t.uidOrganizationId, 
	t.dtmTransactionDate, t.chvTransactionDescription, t.intTransactionTypeId, t.inbFromPartyId, t.inbToPartyId, t.chvPostedViaKey, 
	t.dtmPostedViaDate, y.chvTransactionType
	FROM tb_GLTransaction AS t
	JOIN tb_GLTransactionType AS y
	ON t.intTransactionTypeId = y.intTransactionTypeId
	WHERE t.uidOrganizationId = ? AND t.inbMserviceId = ? AND t.dtmTransactionDate >= ? AND t.dtmTransactionDate <= ?
	AND t.bitIsDeleted = 0 AND y.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	start_date := req.GetStartDate().TimeFromDateTime()
	end_date := req.GetEndDate().TimeFromDateTime()

	rows, err := stmt.Query(req.GetOrganizationId().Guid, req.GetMserviceId(), start_date, end_date)

	if err != nil {
		level.Error(s.logger).Log("what", "Query", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()

	wrapMap := make(map[int64]*pb.GLTransactionWrapper)

	for rows.Next() {
		var from_party sql.NullInt64
		var to_party sql.NullInt64
		var via_key sql.NullString
		var via_date sql.NullTime
		var tran pb.GLTransaction
		var created time.Time
		var modified time.Time
		var trandate time.Time
		var orgGid []byte

		err := rows.Scan(&tran.GlTransactionId, &created, &modified, &tran.Version,
			&tran.MserviceId, &orgGid, &trandate, &tran.TransactionDescription, &tran.TransactionTypeId, &from_party, &to_party, &via_key,
			&via_date, &tran.TransactionType)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		var oid dml.Guid
		oid.Guid = orgGid
		tran.OrganizationId = &oid
		tran.Created = dml.DateTimeFromTime(created)
		tran.Modified = dml.DateTimeFromTime(modified)
		tran.TransactionDate = dml.DateTimeFromTime(trandate)
		if from_party.Valid {
			tran.FromPartyId = from_party.Int64
		}
		if to_party.Valid {
			tran.ToPartyId = to_party.Int64
		}
		if via_key.Valid {
			tran.PostedViaKey = via_key.String
		}

		if via_date.Valid {
			tran.PostedViaDate = dml.DateTimeFromTime(via_date.Time)
		}

		wrap := ConvertTransactionToWrapper(&tran)
		wrapMap[wrap.GetGlTransactionId()] = wrap
		resp.GlTransactionWrappers = append(resp.GlTransactionWrappers, wrap)

	}

	// now get the details
	sqlstring2 := `SELECT d.inbGlTransactionId, d.intSequenceNumber, d.uidGlAccountId, d.decAmount, d.bitIsDebit
	FROM tb_GLTransaction AS t
	JOIN tb_GLTransactionDetail AS d
	ON t.inbGlTransactionId = d.inbGlTransactionId
	WHERE t.uidOrganizationId = ? AND t.inbMserviceId = ? AND t.dtmTransactionDate >= ? AND t.dtmTransactionDate <= ?
	AND t.bitIsDeleted = 0 
	ORDER BY d.inbGlTransactionId, d.intSequenceNumber`

	stmt2, err := s.db.Prepare(sqlstring2)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt2.Close()

	rows2, err := stmt2.Query(req.GetOrganizationId().Guid, req.GetMserviceId(), start_date, end_date)

	if err != nil {
		level.Error(s.logger).Log("what", "Query", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows2.Close()

	for rows2.Next() {
		var gid []byte
		var amount string
		var detail pb.GLTransactionDetail

		err := rows2.Scan(&detail.GlTransactionId, &detail.SequenceNumber, &gid, &amount, &detail.IsDebit)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		wrap, ok := wrapMap[detail.GetGlTransactionId()]
		if ok {
			detail.GlAccountId, _ = dml.GuidFromBytes(gid)
			amt, err := dml.DecimalFromString(amount)
			if err == nil {
				detail.Amount = amt
			}

			wrap.GlTransactionDetails = append(wrap.GlTransactionDetails, &detail)
		}

	}

	return resp, nil
}

// add transaction details
func (s *glService) AddTransactionDetails(ctx context.Context, req *pb.AddTransactionDetailsRequest) (*pb.AddTransactionDetailsResponse, error) {
	resp := &pb.AddTransactionDetailsResponse{}

	// make sure we are referring to a valid transaction
	sqlstring1 := `SELECT inbGlTransactionId FROM tb_GLTransaction WHERE inbGlTransactionId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt1, err := s.db.Prepare(sqlstring1)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt1.Close()

	var transactionId int64

	err = stmt1.QueryRow(req.GetGlTransactionId(), req.GetMserviceId()).Scan(&transactionId)

	if err != nil {
		resp.ErrorCode = 404
		resp.ErrorMessage = "transaction not found"
		return resp, nil
	}

	// make sure the details refer to valid accounts

	sqlstring2 := `SELECT uidGlAccountId FROM tb_GLAccount WHERE uidGlAccountId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`
	stmt2, err := s.db.Prepare(sqlstring2)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt2.Close()

	creditAmt := sdec.New(0, 1) // zero
	debitAmt := sdec.New(0, 1)  // zero
	for _, detail := range req.GetGlTransactionDetails() {
		var accountId []byte

		amt, _ := detail.Amount.ConvertDecimal()

		if detail.IsDebit {
			debitAmt = debitAmt.Add(amt)
		} else {
			creditAmt = creditAmt.Add(amt)
		}

		aid := detail.GetGlAccountId().Guid
		// aid_str := hex.EncodeToString(aid)
		// s.logger.Printf("account_id : %s\n", aid_str)

		err := stmt2.QueryRow(aid, req.GetMserviceId()).Scan(&accountId)
		if err != nil {
			resp.ErrorCode = 404
			resp.ErrorMessage = "account not found"
			return resp, nil
		}
	}

	// make sure debits and credits match
	if !creditAmt.Equals(debitAmt) {
		resp.ErrorCode = 501
		resp.ErrorMessage = "credits and debits do not balance"
		return resp, nil
	}

	err = s.CommitTransactionDetails(req.GetGlTransactionDetails())

	if err != nil {
		resp.ErrorCode = 501
		resp.ErrorMessage = fmt.Sprintf("commit error: %s", err)
	}

	return resp, nil
}

// get current server version and uptime - health check
func (s *glService) GetServerVersion(ctx context.Context, req *pb.GetServerVersionRequest) (*pb.GetServerVersionResponse, error) {
	resp := &pb.GetServerVersionResponse{}

	currentSecs := time.Now().Unix()
	resp.ServerVersion = "v0.9.2"
	resp.ServerUptime = currentSecs - s.startSecs

	return resp, nil
}

func (s *glService) CommitTransactionDetails(details []*pb.GLTransactionDetail) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback() // The rollback will be ignored if the tx has been committed later in the function.

	sqlstring := `INSERT INTO tb_GLTransactionDetail (inbGlTransactionId, intSequenceNumber, uidGlAccountId, decAmount, bitIsDebit)
	VALUES(?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(sqlstring)
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, detail := range details {
		_, err := stmt.Exec(detail.GetGlTransactionId(), detail.GetSequenceNumber(), detail.GetGlAccountId().Guid, detail.GetAmount().StringFromDecimal(),
			detail.GetIsDebit())
		if err != nil {
			return err
		}
	}

	err = tx.Commit()

	return err
}

func (s *glService) GetTransactionHelper(transactionId int64, mserviceId int64) (*genericResponse, *pb.GLTransaction) {
	resp := &genericResponse{}

	sqlstring := `SELECT t.inbGlTransactionId, t.dtmCreated, t.dtmModified, t.intVersion, t.inbMserviceId, t.uidOrganizationId, 
	t.dtmTransactionDate, t.chvTransactionDescription, t.intTransactionTypeId, t.inbFromPartyId, t.inbToPartyId, t.chvPostedViaKey, 
	t.dtmPostedViaDate, y.chvTransactionType
	FROM tb_GLTransaction AS t
	JOIN tb_GLTransactionType AS y
	ON t.intTransactionTypeId = y.intTransactionTypeId
	WHERE t.inbGlTransactionId = ? AND t.inbMserviceId = ? AND t.bitIsDeleted = 0 AND y.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var from_party sql.NullInt64
	var to_party sql.NullInt64
	var via_key sql.NullString
	var via_date sql.NullTime
	var tran pb.GLTransaction
	var created time.Time
	var modified time.Time
	var trandate time.Time
	var orgGid []byte

	err = stmt.QueryRow(transactionId, mserviceId).Scan(&tran.GlTransactionId, &created, &modified, &tran.Version,
		&tran.MserviceId, &orgGid, &trandate, &tran.TransactionDescription, &tran.TransactionTypeId, &from_party, &to_party, &via_key,
		&via_date, &tran.TransactionType)

	if err == nil {
		var oid dml.Guid
		oid.Guid = orgGid
		tran.OrganizationId = &oid
		tran.Created = dml.DateTimeFromTime(created)
		tran.Modified = dml.DateTimeFromTime(modified)
		tran.TransactionDate = dml.DateTimeFromTime(trandate)
		if from_party.Valid {
			tran.FromPartyId = from_party.Int64
		}
		if to_party.Valid {
			tran.ToPartyId = to_party.Int64
		}
		if via_key.Valid {
			tran.PostedViaKey = via_key.String
		}

		if via_date.Valid {
			tran.PostedViaDate = dml.DateTimeFromTime(via_date.Time)
		}

	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, &tran

}

func ConvertTransactionToWrapper(tran *pb.GLTransaction) *pb.GLTransactionWrapper {
	wrap := pb.GLTransactionWrapper{}
	wrap.GlTransactionId = tran.GetGlTransactionId()
	wrap.Created = tran.GetCreated()
	wrap.Modified = tran.GetModified()
	wrap.MserviceId = tran.GetMserviceId()
	wrap.OrganizationId = tran.GetOrganizationId()
	wrap.TransactionDate = tran.GetTransactionDate()
	wrap.TransactionDescription = tran.GetTransactionDescription()
	wrap.TransactionTypeId = tran.GetTransactionTypeId()
	wrap.TransactionType = tran.GetTransactionType()
	wrap.FromPartyId = tran.GetFromPartyId()
	wrap.FromPartyName = tran.GetFromPartyName()
	wrap.ToPartyId = tran.GetToPartyId()
	wrap.ToPartyName = tran.GetToPartyName()
	wrap.PostedViaKey = tran.GetPostedViaKey()
	wrap.PostedViaDate = tran.GetPostedViaDate()

	return &wrap
}
