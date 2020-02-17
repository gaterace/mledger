use mledger;

DROP TABLE IF EXISTS tb_GLTransaction;

-- MService general ledger transaction entity
CREATE TABLE tb_GLTransaction
(

    -- general ledger transaction unique identifier
    inbGlTransactionId BIGINT AUTO_INCREMENT NOT NULL,
    -- creation date
    dtmCreated DATETIME NOT NULL,
    -- modification date
    dtmModified DATETIME NOT NULL,
    -- deletion date
    dtmDeleted DATETIME NOT NULL,
    -- has record been deleted?
    bitIsDeleted BOOL NOT NULL,
    -- version of this record
    intVersion INT NOT NULL,
    -- MService account id
    inbMserviceId BIGINT NOT NULL,
    -- organization unique identifier
    uidOrganizationId BINARY(16) NOT NULL,
    -- transaction date
    dtmTransactionDate DATETIME NOT NULL,
    -- transaction description
    chvTransactionDescription VARCHAR(255) NOT NULL,
    -- general ledger transaction type identifier
    intTransactionTypeId INT NOT NULL,
    -- identifier of transaction from party
    inbFromPartyId BIGINT NULL,
    -- identifier of transaction to party
    inbToPartyId BIGINT NULL,
    -- associated key from external system
    chvPostedViaKey VARCHAR(64) NULL,
    -- date posted on external system
    dtmPostedViaDate DATETIME NULL,


    PRIMARY KEY (inbGlTransactionId),
    INDEX (uidOrganizationId,dtmTransactionDate)
) ENGINE=InnoDB;

