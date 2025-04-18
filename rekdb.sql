SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='TRADITIONAL,ALLOW_INVALID_DATES';

CREATE SCHEMA IF NOT EXISTS `rekdb` ;
USE `rekdb` ;

-- -----------------------------------------------------
-- Table `rekdb`.`job`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdb`.`job` (
  `jid` INT NOT NULL ,
  `jname` VARCHAR(100) NOT NULL ,
  `jtype` TINYINT NULL ,
  `jbelong` INT NULL ,
  `jcreatetime` DATETIME NULL ,
  `jcreateperson` VARCHAR(45) NOT NULL ,
  `jtaskcount` INT NULL ,
  `jlasteditp` VARCHAR(45) NULL ,
  `jgrantps` VARCHAR(200) NULL ,
  `jexecmod` TINYINT NULL ,
  `jexecips` VARCHAR(15000) NULL ,
  `jexecaccount` VARCHAR(45) NULL ,
  `jcronrule` VARCHAR(100) NULL ,
  `jcrondesc` VARCHAR(100) NULL ,
  `jdescrip` VARCHAR(200) NULL ,
  PRIMARY KEY (`jname`, `jcreateperson`) ,
  INDEX `jbelongs` (`jbelong` ASC) ,
  INDEX `jcreatep` (`jcreateperson` ASC) ,
  UNIQUE INDEX `jid_UNIQUE` (`jid` ASC) )
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`jobrunhistory`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdbrun`.`jobrunhistory` (
  `runid` INT NOT NULL ,
  `jid` INT NOT NULL ,
  `jname` VARCHAR(100) NOT NULL ,
  `jtype` TINYINT NULL ,
  `jbelong` INT NULL ,
  `jcreatetime` DATETIME NULL ,
  `jcreateperson` VARCHAR(45) NOT NULL ,
  `jtaskcount` INT NULL ,
  `jlasteditp` VARCHAR(45) NULL ,
  `jgrantps` VARCHAR(200) NULL ,
  `jexecmod` TINYINT NULL ,
  `jexecips` VARCHAR(15000) NULL ,
  `jexecaccount` VARCHAR(45) NULL ,
  `jcronrule` VARCHAR(100) NULL ,
  `jcrondesc` VARCHAR(100) NULL ,
  `jdescrip` VARCHAR(200) NULL ,
  INDEX `jbelongs` (`jbelong` ASC) ,
  INDEX `jcreatep` (`jcreateperson` ASC) ,
  PRIMARY KEY (`runid`, `jid`) )
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`responserun`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdbrun`.`responserun` (
  `runid` INT NOT NULL ,
  `jid` INT NOT NULL ,
  `tid` INT NOT NULL ,
  `agentip` VARCHAR(45) NOT NULL ,
  `flag` TINYINT(1) NULL ,
  `result` VARCHAR(1000) NULL ,
  INDEX `ind` (`flag` ASC) ,
  PRIMARY KEY (`runid`, `jid`, `tid`, `agentip`) ,
  INDEX `r44id_idx` (`runid` ASC) ,
  CONSTRAINT `r44id`
    FOREIGN KEY (`runid` )
    REFERENCES `rekdbrun`.`jobrunhistory` (`runid` )
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`M_script`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdb`.`M_script` (
  `sid` INT NOT NULL ,
  `scriptfilename` VARCHAR(45) NOT NULL ,
  `scriptdesc` VARCHAR(45) NULL ,
  `createperson` VARCHAR(45) NOT NULL ,
  `lasteditpers` VARCHAR(45) NULL ,
  `grantpers` VARCHAR(200) NULL ,
  `createtime` DATETIME NULL ,
  `status` TINYINT NULL ,
  UNIQUE INDEX `sid_UNIQUE` (`sid` ASC) ,
  PRIMARY KEY (`scriptfilename`, `createperson`) )
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`M_account`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdb`.`M_account` (
  `aid` INT NOT NULL ,
  `accountname` VARCHAR(45) NOT NULL ,
  `accountalias` VARCHAR(45) NOT NULL ,
  `createtime` DATETIME NULL ,
  `createperson` VARCHAR(45) NOT NULL ,
  `lasteditpers` VARCHAR(45) NULL ,
  `lastedittime` DATETIME NULL ,
  `passwd` VARCHAR(100) NOT NULL ,
  UNIQUE INDEX `accountalias_UNIQUE` (`accountalias` ASC) ,
  UNIQUE INDEX `aid_UNIQUE` (`aid` ASC) ,
  PRIMARY KEY (`createperson`, `accountalias`) )
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`M_file`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdb`.`M_file` (
  `fid` INT NOT NULL ,
  `fname` VARCHAR(45) NOT NULL ,
  `furl` VARCHAR(100) NOT NULL ,
  `fmd5` VARCHAR(45) NOT NULL ,
  `fbelongpers` VARCHAR(45) NOT NULL ,
  PRIMARY KEY (`fname`, `fbelongpers`) ,
  UNIQUE INDEX `fmd5_UNIQUE` (`fmd5` ASC) ,
  UNIQUE INDEX `fid_UNIQUE` (`fid` ASC) )
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`task`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdb`.`task` (
  `jid` INT NOT NULL ,
  `tid` INT NOT NULL ,
  `ttype` TINYINT NOT NULL ,
  `tname` VARCHAR(45) NOT NULL ,
  `tdesc` VARCHAR(45) NULL ,
  `texecaccount` INT NULL ,
  `texecips` VARCHAR(15000) NULL ,
  `tscriptfilename` INT NULL ,
  `tscriptargs` VARCHAR(45) NULL ,
  `tscripttimeout` INT NULL ,
  `tfilesrcfile` INT NULL ,
  `tfiledstpath` VARCHAR(45) NULL ,
  `tfilefrom` TINYINT NULL ,
  `techocontent` VARCHAR(100) NULL ,
  PRIMARY KEY (`jid`, `tid`) ,
  INDEX `tsfn_idx` (`tscriptfilename` ASC) ,
  INDEX `tacc_idx` (`texecaccount` ASC) ,
  CONSTRAINT `jiddddd`
    FOREIGN KEY (`jid` )
    REFERENCES `rekdb`.`job` (`jid` )
    ON DELETE CASCADE
    ON UPDATE NO ACTION,
  CONSTRAINT `tsfn`
    FOREIGN KEY (`tscriptfilename` )
    REFERENCES `rekdb`.`M_script` (`sid` )
    ON DELETE NO ACTION
    ON UPDATE NO ACTION,
  CONSTRAINT `tfn`
    FOREIGN KEY (`tscriptfilename` )
    REFERENCES `rekdb`.`M_file` (`fid` )
    ON DELETE NO ACTION
    ON UPDATE NO ACTION,
  CONSTRAINT `tacc`
    FOREIGN KEY (`texecaccount` )
    REFERENCES `rekdb`.`M_account` (`aid` )
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`taskrunhistory`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdbrun`.`taskrunhistory` (
  `runid` INT NOT NULL ,
  `jid` INT NOT NULL ,
  `tid` INT NOT NULL ,
  `ttype` TINYINT NOT NULL ,
  `tname` VARCHAR(45) NOT NULL ,
  `tdesc` VARCHAR(45) NULL ,
  `texecaccount` INT NULL ,
  `texecips` VARCHAR(15000) NULL ,
  `tscriptfilename` INT NULL ,
  `tscriptargs` VARCHAR(45) NULL ,
  `tscripttimeout` INT NULL ,
  `tfilesrcfile` INT NULL ,
  `tfiledstpath` VARCHAR(45) NULL ,
  `tfilefrom` TINYINT NULL ,
  `techocontent` VARCHAR(100) NULL ,
  PRIMARY KEY (`runid`) ,
  CONSTRAINT `r22id`
    FOREIGN KEY (`runid` )
    REFERENCES `rekdbrun`.`jobrunhistory` (`runid` )
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`jobrun`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdbrun`.`jobrun` (
  `runid` INT NOT NULL ,
  `runstatus` TINYINT NULL ,
  `runstart` DATETIME NULL ,
  `runstop` DATETIME NULL ,
  `runtime` INT NULL ,
  `runperson` VARCHAR(45) NULL ,
  PRIMARY KEY (`runid`) ,
  CONSTRAINT `j1`
    FOREIGN KEY (`runid` )
    REFERENCES `rekdbrun`.`jobrunhistory` (`runid` )
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `rekdb`.`taskrun`
-- -----------------------------------------------------
CREATE  TABLE IF NOT EXISTS `rekdbrun`.`taskrun` (
  `runid` INT NOT NULL ,
  `tid` INT NOT NULL ,
  `runstatus` TINYINT NULL ,
  `runstart` DATETIME NULL ,
  `runstop` DATETIME NULL ,
  `runtime` INT NULL ,
  PRIMARY KEY (`runid`, `tid`) ,
  INDEX `j2_idx` (`runid` ASC) ,
  CONSTRAINT `j2`
    FOREIGN KEY (`runid` )
    REFERENCES `rekdbrun`.`taskrunhistory` (`runid` )
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;

USE `rekdb` ;
USE `rekdbrun`;

DELIMITER $$

DELIMITER ;


SET SQL_MODE=@OLD_SQL_MODE;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
