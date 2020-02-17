use mledger;

DROP TABLE IF EXISTS tb_GLTransactionType;

-- MService general ledger transaction type entity
CREATE TABLE tb_GLTransactionType
(

    -- MService account id
    inbMserviceId BIGINT NOT NULL,
    -- general ledger transaction type identifier
    intTransactionTypeId INT NOT NULL,
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
    -- transaction type description
    chvTransactionType VARCHAR(255) NOT NULL,


    PRIMARY KEY (inbMserviceId,intTransactionTypeId)
) ENGINE=InnoDB;

