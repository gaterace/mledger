use mledger;

DROP TABLE IF EXISTS tb_GLTransactionDetail;

-- MService general ledger transaction detail entity
CREATE TABLE tb_GLTransactionDetail
(

    -- general ledger transaction unique identifier
    inbGlTransactionId BIGINT NOT NULL,
    -- transaction detail sequence number
    intSequenceNumber INT NOT NULL,
    -- general ledger account unique identifier
    uidGlAccountId BINARY(16) NOT NULL,
    -- transaction detail amount
    decAmount DECIMAL(19,2) NOT NULL,
    -- transaction detail is debit (true) else credit
    bitIsDebit BOOL NOT NULL,


    PRIMARY KEY (inbGlTransactionId,intSequenceNumber)
) ENGINE=InnoDB;

