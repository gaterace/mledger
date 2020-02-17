use mledger;

DROP TABLE IF EXISTS tb_GLOrganization;

-- MService general ledger organization entity
CREATE TABLE tb_GLOrganization
(

    -- organization unique identifier
    uidOrganizationId BINARY(16) NOT NULL,
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
    -- organization name
    chvOrganizationName VARCHAR(32) NOT NULL,
    -- starting date for organization books
    dtmFromDate DATETIME NOT NULL,
    -- ending date for organization books
    dtmToDate DATETIME NULL,


    PRIMARY KEY (uidOrganizationId),
    UNIQUE (inbMserviceId,chvOrganizationName)
) ENGINE=InnoDB;

