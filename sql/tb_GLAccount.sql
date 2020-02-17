use mledger;

DROP TABLE IF EXISTS tb_GLAccount;

-- MService general ledger account entity
CREATE TABLE tb_GLAccount
(

    -- general ledger account unique identifier
    uidGlAccountId BINARY(16) NOT NULL,
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
    -- general ledger account name
    chvAccountName VARCHAR(32) NOT NULL,
    -- general ledger account description
    chvAccountDescription VARCHAR(255) NOT NULL,
    -- general ledger account type identifier
    intAccountTypeId INT NOT NULL,


    PRIMARY KEY (uidGlAccountId),
    UNIQUE (uidOrganizationId,chvAccountName)
) ENGINE=InnoDB;

