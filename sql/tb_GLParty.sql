use mledger;

DROP TABLE IF EXISTS tb_GLParty;

-- MService general ledger transaction party entity
CREATE TABLE tb_GLParty
(

    -- MService account id
    inbMserviceId BIGINT NOT NULL,
    -- transaction party identifier
    inbPartyId BIGINT NOT NULL,
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
    -- transaction party name
    chvPartyName VARCHAR(255) NOT NULL,


    PRIMARY KEY (inbMserviceId,inbPartyId)
) ENGINE=InnoDB;

