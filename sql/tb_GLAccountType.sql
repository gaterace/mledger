use mledger;

DROP TABLE IF EXISTS tb_GLAccountType;

-- MService general ledger account type entity
CREATE TABLE tb_GLAccountType
(

    -- MService account id
    inbMserviceId BIGINT NOT NULL,
    -- general ledger account type identifier
    intAccountTypeId INT NOT NULL,
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
    -- general ledger account type
    chvAccountType VARCHAR(255) NOT NULL,


    PRIMARY KEY (inbMserviceId,intAccountTypeId)
) ENGINE=InnoDB;

