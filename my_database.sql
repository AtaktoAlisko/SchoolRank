-- MySQL dump 10.13  Distrib 9.1.0, for macos14 (arm64)
--
-- Host: localhost    Database: my_database
-- ------------------------------------------------------
-- Server version	9.1.0

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `City_Olympiad`
--

DROP TABLE IF EXISTS `City_Olympiad`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `City_Olympiad` (
  `id` int NOT NULL AUTO_INCREMENT,
  `student_id` int DEFAULT NULL,
  `city_olympiad_place` int DEFAULT NULL,
  `score` int DEFAULT NULL,
  `competition_date` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `school_id` int DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `student_id` (`student_id`),
  CONSTRAINT `city_olympiad_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `Student` (`student_id`)
) ENGINE=InnoDB AUTO_INCREMENT=26 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `City_Olympiad`
--

LOCK TABLES `City_Olympiad` WRITE;
/*!40000 ALTER TABLE `City_Olympiad` DISABLE KEYS */;
INSERT INTO `City_Olympiad` VALUES (13,23,2,30,'2025-03-27 19:00:00',36),(14,23,1,50,'2025-03-27 19:00:00',36),(15,23,3,20,'2025-03-27 19:00:00',36),(16,25,3,20,'2025-03-27 19:00:00',36),(17,25,2,30,'2025-03-27 19:00:00',36),(18,25,1,50,'2025-03-27 19:00:00',36),(19,27,1,50,'2025-03-27 19:00:00',36),(20,27,2,30,'2025-03-27 19:00:00',36),(21,27,3,20,'2025-03-27 19:00:00',36),(22,27,3,20,'2025-03-27 19:00:00',38),(23,29,3,20,'2025-03-27 19:00:00',38),(24,33,1,50,'2025-03-27 19:00:00',38),(25,34,2,30,'2025-03-27 19:00:00',38);
/*!40000 ALTER TABLE `City_Olympiad` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `First_Type`
--

DROP TABLE IF EXISTS `First_Type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `First_Type` (
  `first_type_id` int NOT NULL AUTO_INCREMENT,
  `history_of_kazakhstan` int DEFAULT NULL,
  `mathematical_literacy` int NOT NULL DEFAULT '0',
  `reading_literacy` int NOT NULL DEFAULT '0',
  `first_subject_score` int DEFAULT NULL,
  `second_subject_score` int DEFAULT NULL,
  `school_id` int DEFAULT NULL,
  `total_score` int DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `student_id` int DEFAULT NULL,
  `first_subject` varchar(255) DEFAULT NULL,
  `second_subject` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`first_type_id`),
  KEY `FK_School` (`school_id`),
  CONSTRAINT `FK_School` FOREIGN KEY (`school_id`) REFERENCES `schools` (`school_id`)
) ENGINE=InnoDB AUTO_INCREMENT=56 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `First_Type`
--

LOCK TABLES `First_Type` WRITE;
/*!40000 ALTER TABLE `First_Type` DISABLE KEYS */;
INSERT INTO `First_Type` VALUES (1,20,25,15,30,30,NULL,NULL,NULL,NULL,NULL,NULL),(9,20,10,10,30,30,NULL,NULL,NULL,NULL,NULL,NULL),(11,20,10,10,30,30,NULL,NULL,NULL,NULL,NULL,NULL),(19,12,8,8,25,28,NULL,NULL,NULL,NULL,NULL,NULL),(20,0,10,10,30,30,NULL,NULL,NULL,NULL,NULL,NULL),(21,0,10,10,38,36,NULL,NULL,NULL,NULL,NULL,NULL),(22,0,10,10,38,36,NULL,NULL,NULL,NULL,NULL,NULL),(23,0,10,10,38,36,NULL,NULL,NULL,NULL,NULL,NULL),(24,0,10,10,38,36,NULL,NULL,NULL,NULL,NULL,NULL),(25,0,10,10,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL),(26,12,10,10,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL),(27,12,10,10,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL),(28,12,10,10,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL),(29,12,10,10,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL),(30,12,10,10,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL),(31,12,10,10,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL),(32,12,10,10,NULL,NULL,24,NULL,NULL,NULL,NULL,NULL),(33,12,10,10,NULL,NULL,24,32,NULL,NULL,NULL,NULL),(34,12,10,10,NULL,NULL,24,112,NULL,NULL,NULL,NULL),(35,10,10,10,NULL,NULL,24,90,NULL,NULL,NULL,NULL),(36,10,10,10,NULL,NULL,29,90,NULL,NULL,NULL,NULL),(37,11,11,11,NULL,NULL,29,93,NULL,NULL,NULL,NULL),(38,15,10,10,NULL,NULL,29,113,NULL,NULL,NULL,NULL),(39,15,10,10,NULL,NULL,29,NULL,NULL,NULL,NULL,NULL),(40,15,10,10,NULL,NULL,29,NULL,NULL,NULL,NULL,NULL),(41,15,10,10,NULL,NULL,29,NULL,NULL,NULL,NULL,NULL),(42,15,10,10,NULL,NULL,29,NULL,NULL,NULL,NULL,NULL),(43,15,10,10,NULL,NULL,29,NULL,'type-1',NULL,NULL,NULL),(44,15,10,10,NULL,NULL,29,NULL,'type-1',NULL,NULL,NULL),(48,10,5,8,NULL,NULL,40,NULL,'type-1',NULL,NULL,NULL),(49,10,5,8,NULL,NULL,40,NULL,'type-1',NULL,NULL,NULL),(50,10,5,8,NULL,NULL,40,NULL,'type-1',NULL,NULL,NULL),(51,10,5,8,NULL,NULL,40,NULL,'type-1',NULL,NULL,NULL),(52,10,14,13,12,15,NULL,NULL,'type-1',37,'Mathematics','Physics'),(53,10,14,13,12,15,41,NULL,'type-1',38,'Mathematics','Physics'),(54,10,14,13,12,15,41,64,'type-1',39,'Mathematics','Physics'),(55,8,2,10,30,45,41,95,'type-1',41,'Mathematics','Physics');
/*!40000 ALTER TABLE `First_Type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `password_resets`
--

DROP TABLE IF EXISTS `password_resets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `password_resets` (
  `id` int NOT NULL AUTO_INCREMENT,
  `email` varchar(255) NOT NULL,
  `otp_code` varchar(6) DEFAULT NULL,
  `reset_token` varchar(255) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=45 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `password_resets`
--

LOCK TABLES `password_resets` WRITE;
/*!40000 ALTER TABLE `password_resets` DISABLE KEYS */;
INSERT INTO `password_resets` VALUES (26,'erkinbz@gmail.com','109009','KjxFt8B_27s-MzxBct1teMsTOu_apeaulX2iwbrXPkM=','2025-02-18 22:56:18');
/*!40000 ALTER TABLE `password_resets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `Regional_Olympiad`
--

DROP TABLE IF EXISTS `Regional_Olympiad`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `Regional_Olympiad` (
  `id` int NOT NULL AUTO_INCREMENT,
  `student_id` int DEFAULT NULL,
  `regional_olympiad_place` int DEFAULT NULL,
  `score` int DEFAULT NULL,
  `competition_date` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `school_id` int DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `student_id` (`student_id`),
  CONSTRAINT `regional_olympiad_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `Student` (`student_id`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `Regional_Olympiad`
--

LOCK TABLES `Regional_Olympiad` WRITE;
/*!40000 ALTER TABLE `Regional_Olympiad` DISABLE KEYS */;
INSERT INTO `Regional_Olympiad` VALUES (6,23,3,20,'2025-03-27 23:38:12',36),(7,23,2,30,'2025-03-27 23:38:27',36),(8,23,1,50,'2025-03-27 23:38:37',36),(9,23,1,50,'2025-03-27 23:53:56',36),(10,23,1,50,'2025-03-28 00:03:00',36);
/*!40000 ALTER TABLE `Regional_Olympiad` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `Republican_Olympiad`
--

DROP TABLE IF EXISTS `Republican_Olympiad`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `Republican_Olympiad` (
  `id` int NOT NULL AUTO_INCREMENT,
  `student_id` int NOT NULL,
  `republican_olympiad_place` int NOT NULL,
  `score` int DEFAULT NULL,
  `competition_date` datetime DEFAULT NULL,
  `school_id` int DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `student_id` (`student_id`),
  KEY `school_id` (`school_id`),
  CONSTRAINT `republican_olympiad_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `Student` (`student_id`),
  CONSTRAINT `republican_olympiad_ibfk_2` FOREIGN KEY (`school_id`) REFERENCES `Schools` (`school_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `Republican_Olympiad`
--

LOCK TABLES `Republican_Olympiad` WRITE;
/*!40000 ALTER TABLE `Republican_Olympiad` DISABLE KEYS */;
/*!40000 ALTER TABLE `Republican_Olympiad` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `RepublicanOlympiad`
--

DROP TABLE IF EXISTS `RepublicanOlympiad`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `RepublicanOlympiad` (
  `id` int NOT NULL AUTO_INCREMENT,
  `student_id` int DEFAULT NULL,
  `republican_olympiad_place` int DEFAULT NULL,
  `score` int DEFAULT NULL,
  `competition_date` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `school_id` int DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `student_id` (`student_id`),
  CONSTRAINT `republicanolympiad_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `Student` (`student_id`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `RepublicanOlympiad`
--

LOCK TABLES `RepublicanOlympiad` WRITE;
/*!40000 ALTER TABLE `RepublicanOlympiad` DISABLE KEYS */;
INSERT INTO `RepublicanOlympiad` VALUES (4,27,2,30,'2025-03-28 00:17:42',36),(5,23,2,30,'2025-03-28 00:34:08',36),(6,23,1,50,'2025-03-28 00:34:11',36),(7,27,3,20,'2025-03-28 00:34:16',36);
/*!40000 ALTER TABLE `RepublicanOlympiad` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `Reviews`
--

DROP TABLE IF EXISTS `Reviews`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `Reviews` (
  `id` int NOT NULL AUTO_INCREMENT,
  `school_id` int DEFAULT NULL,
  `user_id` int DEFAULT NULL,
  `rating` float DEFAULT NULL,
  `comment` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `school_id` (`school_id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `reviews_ibfk_1` FOREIGN KEY (`school_id`) REFERENCES `Schools` (`school_id`),
  CONSTRAINT `reviews_ibfk_2` FOREIGN KEY (`user_id`) REFERENCES `Users` (`id`),
  CONSTRAINT `reviews_chk_1` CHECK ((`rating` between 1 and 5))
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `Reviews`
--

LOCK TABLES `Reviews` WRITE;
/*!40000 ALTER TABLE `Reviews` DISABLE KEYS */;
INSERT INTO `Reviews` VALUES (1,36,3,5,'Отличная школа, преподаватели очень отзывчивые!','2025-03-27 18:15:48'),(2,36,9,4,'Отличная школа, преподаватели очень отзывчивые!','2025-03-27 18:16:31'),(3,36,10,2,'Отличная школа, преподаватели очень отзывчивые!','2025-03-27 18:16:38'),(5,38,12,4,'Great school, highly recommend!','2025-03-31 17:44:36'),(6,38,12,4,'Great school, highly recommend!','2025-03-31 17:44:44'),(7,38,12,3,'Great school, highly recommend!','2025-03-31 17:45:04');
/*!40000 ALTER TABLE `Reviews` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `Schools`
--

DROP TABLE IF EXISTS `Schools`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `Schools` (
  `school_id` int NOT NULL AUTO_INCREMENT,
  `user_id` int DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `address` text NOT NULL,
  `title` varchar(255) NOT NULL,
  `description` text NOT NULL,
  `contacts` text NOT NULL,
  `photo_url` varchar(255) NOT NULL,
  `republican_olympiad_rating` float DEFAULT NULL,
  PRIMARY KEY (`school_id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `fk_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `schools_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=42 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `Schools`
--

LOCK TABLES `Schools` WRITE;
/*!40000 ALTER TABLE `Schools` DISABLE KEYS */;
INSERT INTO `Schools` VALUES (13,NULL,'','','','','','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-photo.jpg',NULL),(15,NULL,'Школа №1','г. Алматы, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-photo.jpg',NULL),(16,NULL,'Школа №1','г. Алматы, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-photo.jpg',NULL),(17,NULL,'Школа №1','г. Алматы, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-photo.jpg',NULL),(18,NULL,'Школа №1','г. Алматы, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-27-1742120096.jpg',NULL),(19,NULL,'Школа №1','г. Алматы, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-27-1742120977.jpg',NULL),(21,NULL,'Школа №1','г. Алматы, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-27-1742161691.jpg',NULL),(24,NULL,'Школа №1','г. Алматы, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-27-1742292300.jpg',NULL),(29,27,'Школа №20','г. Астана, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-27-1742595330.jpg',NULL),(35,29,'Школа №200','г. Астана, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-29-1742764071.jpg',NULL),(36,28,'Школа №200','г. Астана, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-28-1742765536.jpg',0.325),(37,31,'Школа №200','г. Астана, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-31-1743442656.jpg',NULL),(38,31,'Школа №200','г. Астана, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-31-1743442744.jpg',NULL),(39,31,'Школа №200','г. Астана, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-31-1743541067.jpg',NULL),(40,39,'Школа №200','г. Астана, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-39-1743660290.jpg',NULL),(41,39,'Школа №200','г. Астана, ул. Абая, 10','Одна из лучших школ','Школа с высокими результатами','Телефон: 123456789, Email: info@school.kz','https://schoolrank-schoolphotos.s3.eu-north-1.amazonaws.com/school-39-1743738964.jpg',NULL);
/*!40000 ALTER TABLE `Schools` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `Second_Type`
--

DROP TABLE IF EXISTS `Second_Type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `Second_Type` (
  `second_type_id` int NOT NULL AUTO_INCREMENT,
  `history_of_kazakhstan_creative` int DEFAULT '0',
  `reading_literacy_creative` int DEFAULT NULL,
  `school_id` int DEFAULT NULL,
  `total_score_creative` int DEFAULT NULL,
  `creative_exam1` int DEFAULT '0',
  `creative_exam2` int DEFAULT '0',
  `type` varchar(255) NOT NULL DEFAULT 'type-2',
  `student_id` int DEFAULT NULL,
  PRIMARY KEY (`second_type_id`),
  KEY `fk_school_id` (`school_id`),
  CONSTRAINT `fk_school_id` FOREIGN KEY (`school_id`) REFERENCES `schools` (`school_id`)
) ENGINE=InnoDB AUTO_INCREMENT=26 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `Second_Type`
--

LOCK TABLES `Second_Type` WRITE;
/*!40000 ALTER TABLE `Second_Type` DISABLE KEYS */;
INSERT INTO `Second_Type` VALUES (2,20,10,NULL,NULL,0,0,'type-2',NULL),(3,20,10,NULL,NULL,0,0,'type-2',NULL),(4,20,10,NULL,NULL,0,0,'type-2',NULL),(5,12,10,NULL,NULL,0,0,'type-2',NULL),(6,8,8,NULL,NULL,0,0,'type-2',NULL),(7,8,8,NULL,NULL,0,0,'type-2',NULL),(8,7,7,NULL,NULL,0,0,'type-2',NULL),(9,10,10,NULL,NULL,0,0,'type-2',NULL),(10,10,10,NULL,NULL,0,0,'type-2',NULL),(11,12,10,NULL,NULL,0,0,'type-2',NULL),(12,12,10,NULL,NULL,0,0,'type-2',NULL),(13,12,10,24,NULL,0,0,'type-2',NULL),(14,10,10,24,NULL,0,0,'type-2',NULL),(15,10,10,29,NULL,0,0,'type-2',NULL),(16,20,10,29,120,45,45,'type-2',NULL),(17,20,10,29,75,35,10,'type-2',NULL),(18,20,10,29,75,35,10,'type-2',NULL),(19,20,10,29,75,35,10,'type-2',NULL),(20,20,10,29,75,35,10,'creative',NULL),(21,20,10,29,75,35,10,'creative',NULL),(22,20,10,29,75,35,10,'type-2',NULL),(23,10,12,41,52,14,16,'type-2',41),(24,10,12,41,52,14,16,'type-2',NULL),(25,10,12,41,52,14,16,'type-2',39);
/*!40000 ALTER TABLE `Second_Type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `student`
--

DROP TABLE IF EXISTS `student`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `student` (
  `student_id` int NOT NULL AUTO_INCREMENT,
  `first_name` varchar(100) NOT NULL,
  `last_name` varchar(100) NOT NULL,
  `patronymic` varchar(100) DEFAULT NULL,
  `iin` varchar(12) NOT NULL,
  `school_id` int DEFAULT NULL,
  `date_of_birth` date DEFAULT NULL,
  `grade` int DEFAULT NULL,
  `letter` varchar(1) DEFAULT NULL,
  `gender` varchar(10) DEFAULT NULL,
  `phone` varchar(15) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`student_id`),
  UNIQUE KEY `iin` (`iin`)
) ENGINE=InnoDB AUTO_INCREMENT=42 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `student`
--

LOCK TABLES `student` WRITE;
/*!40000 ALTER TABLE `student` DISABLE KEYS */;
INSERT INTO `student` VALUES (7,'Ali','Bek','Murad','092992993',NULL,NULL,NULL,NULL,NULL,NULL,NULL),(8,'John','Doe','Alexeyevich','1234567890',NULL,NULL,NULL,NULL,NULL,NULL,NULL),(9,'Zhasulan','Kaurat','Nursultanovich','9299232939',6,NULL,NULL,NULL,NULL,NULL,NULL),(10,'Murat','Alibek','Nursultanovich','9299232999',6,NULL,NULL,NULL,NULL,NULL,NULL),(11,'John','Doe','Smith','123456789012',6,NULL,NULL,NULL,NULL,NULL,NULL),(16,'Mura','Alish','Smith','123456883012',24,'2005-06-15',11,NULL,NULL,NULL,NULL),(17,'Murat','Alibek','KKK','27090450141',29,'2005-06-15',11,NULL,NULL,NULL,NULL),(19,'Alinur ','Shayakhmet','KKK','2709045141',29,'2005-06-15',11,NULL,NULL,NULL,NULL),(20,'Laura ','Kazimhanova','KKK','2709099141',29,'2005-06-15',11,NULL,NULL,NULL,NULL),(23,'Test1 ','Test1','KKK','2709099241',36,'2005-06-15',11,NULL,NULL,NULL,NULL),(25,'Test2 ','Test2','KKK','2709099240',36,'2005-06-15',11,NULL,NULL,NULL,NULL),(27,'Test3 ','Test3','KKK','270909941',36,'2005-06-15',11,NULL,NULL,NULL,NULL),(29,'Alinur ','Alinur','Alinur','27090999941',38,'2005-06-15',11,NULL,NULL,NULL,NULL),(33,'Laura ','Laura','Laura','270909',38,'2005-06-15',11,NULL,NULL,NULL,NULL),(34,'Alibek ','Alibek','Alibek','2709034349',38,'2005-06-15',11,NULL,NULL,NULL,NULL),(36,'Murat ','Иванов','Иванович','123456789099',0,'2005-09-15',11,'A','Мale','+77012345678','ivan.ivanov@example.com'),(37,'Alihan ','Alihan','Alihan','1289099',40,'2005-06-15',11,'A','Мale','+77012345678','ivan.ivanov@example.com'),(38,'Dimash ','Dimash','Dimash','12899',40,'2005-06-15',11,'A','Мale','+77012345678','ivan.ivanov@example.com'),(39,'Alibek ','Dimash','Dimash','12899000',40,'2005-06-15',11,'B','Мale','+77012345678','ivan.ivanov@example.com'),(41,'Alibsdsek ','Dimash','Dimash','128900',40,'2005-06-15',11,'B','Мale','+77012345678','ivan.ivanov@example.com');
/*!40000 ALTER TABLE `student` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `student_ratings`
--

DROP TABLE IF EXISTS `student_ratings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `student_ratings` (
  `student_id` int NOT NULL,
  `year` int NOT NULL,
  `total_score` int NOT NULL,
  `first_type_id` int DEFAULT NULL,
  `second_type_id` int DEFAULT NULL,
  `first_subject_name` varchar(255) DEFAULT NULL,
  `first_subject_score` int DEFAULT NULL,
  `second_subject_name` varchar(255) DEFAULT NULL,
  `second_subject_score` int DEFAULT NULL,
  `history_of_kazakhstan` int DEFAULT NULL,
  `reading_literacy` int DEFAULT NULL,
  `rating` float DEFAULT NULL,
  PRIMARY KEY (`student_id`,`year`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `student_ratings`
--

LOCK TABLES `student_ratings` WRITE;
/*!40000 ALTER TABLE `student_ratings` DISABLE KEYS */;
/*!40000 ALTER TABLE `student_ratings` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `Total_Olympiad_Rating`
--

DROP TABLE IF EXISTS `Total_Olympiad_Rating`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `Total_Olympiad_Rating` (
  `school_id` int NOT NULL,
  `total_rating` float DEFAULT NULL,
  PRIMARY KEY (`school_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `Total_Olympiad_Rating`
--

LOCK TABLES `Total_Olympiad_Rating` WRITE;
/*!40000 ALTER TABLE `Total_Olympiad_Rating` DISABLE KEYS */;
/*!40000 ALTER TABLE `Total_Olympiad_Rating` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `UNT_Score`
--

DROP TABLE IF EXISTS `UNT_Score`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `UNT_Score` (
  `unt_score_id` int NOT NULL AUTO_INCREMENT,
  `year` int NOT NULL,
  `unt_type_id` int DEFAULT NULL,
  `student_id` int NOT NULL,
  `first_subject_score` int DEFAULT '0',
  `second_subject_score` int DEFAULT '0',
  `reading_literacy` int DEFAULT '0',
  `math_literacy` int DEFAULT '0',
  `total_score` int DEFAULT NULL,
  `history_of_kazakhstan` int DEFAULT '0',
  `first_subject_name` varchar(255) DEFAULT '',
  `second_subject_name` varchar(255) DEFAULT '',
  `first_name` varchar(255) DEFAULT NULL,
  `last_name` varchar(255) DEFAULT NULL,
  `iin` varchar(12) DEFAULT NULL,
  `score` int DEFAULT NULL,
  `rating` float DEFAULT '0',
  `creative_exam1` int DEFAULT '0',
  `creative_exam2` int DEFAULT '0',
  `total_score_creative` int DEFAULT '0',
  `average_rating` float DEFAULT '0',
  `average_rating_second` float DEFAULT '0',
  PRIMARY KEY (`unt_score_id`),
  KEY `FK_UNT_Type` (`unt_type_id`),
  KEY `FK_Student` (`student_id`),
  CONSTRAINT `FK_Student` FOREIGN KEY (`student_id`) REFERENCES `Student` (`student_id`),
  CONSTRAINT `unt_score_ibfk_1` FOREIGN KEY (`unt_type_id`) REFERENCES `UNT_Type` (`unt_type_id`),
  CONSTRAINT `unt_score_ibfk_2` FOREIGN KEY (`student_id`) REFERENCES `Student` (`student_id`)
) ENGINE=InnoDB AUTO_INCREMENT=42 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `UNT_Score`
--

LOCK TABLES `UNT_Score` WRITE;
/*!40000 ALTER TABLE `UNT_Score` DISABLE KEYS */;
INSERT INTO `UNT_Score` VALUES (5,2025,NULL,7,18,20,2,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,0,0,0,0,0),(9,2025,NULL,9,30,25,25,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,0,0,0,0,0),(10,2025,NULL,9,30,25,25,0,0,15,NULL,NULL,NULL,NULL,NULL,NULL,0,0,0,0,0,0),(11,2025,NULL,9,30,25,25,0,0,15,NULL,NULL,NULL,NULL,NULL,NULL,0,0,0,0,0,0),(12,2025,NULL,9,30,25,25,0,0,15,NULL,NULL,NULL,NULL,NULL,NULL,0,0,0,0,0,0),(13,2025,15,10,NULL,NULL,NULL,NULL,0,NULL,NULL,NULL,NULL,NULL,NULL,NULL,132,0,0,22,0,0),(14,2025,15,10,50,50,8,0,0,8,NULL,NULL,NULL,NULL,NULL,NULL,132,0,0,22,0,0),(15,2025,15,11,50,50,8,0,0,8,NULL,NULL,NULL,NULL,NULL,NULL,132,0,0,22,0,0),(16,2025,15,11,50,50,8,0,0,8,NULL,NULL,NULL,NULL,NULL,NULL,132,0,0,22,0,0),(17,2025,15,11,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,100,132,0,0,22,0,0),(18,2025,15,11,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,100,132,0,0,22,0,0),(19,2025,17,11,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,40,72,0,0,22,0,0),(20,2025,17,11,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,40,72,0,0,22,0,0),(21,2025,15,11,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,132,0,0,22,0,0),(22,2025,15,11,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,132,0,0,22,0,0),(23,2025,17,11,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,40,72,0,0,22,0,0),(24,2025,20,11,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,82,0,0,22,0,0),(25,2025,20,11,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,82,0,0,22,0,0),(26,2025,17,10,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,72,0,0,22,0,0),(27,2025,24,17,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,90,0,0,20,0,0),(28,2025,24,19,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,90,0,0,20,0,0),(29,2025,25,19,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,93,0,0,22,0,0),(30,2025,26,20,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,113,0,0,25,0,0),(31,2025,27,20,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,0,0,0,0,0,0),(32,2025,28,23,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,113,0,0,25,0,0),(33,2025,29,25,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,95,0,0,25,0,0),(34,2025,30,27,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,80,0,0,25,0,0),(35,2025,30,27,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,80,0,0,25,0,0),(36,2025,31,27,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,0,0,0,0,0,0),(37,2025,32,27,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,0,0,0,0,0,0),(38,2025,39,41,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,0,0,0,0,0,0),(39,2025,38,39,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,0,0,0,0,0,0),(40,2025,37,38,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,0,0,0,0,0,0),(41,2025,36,37,0,0,0,0,0,0,'','',NULL,NULL,NULL,NULL,0,0,0,0,0,0);
/*!40000 ALTER TABLE `UNT_Score` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `UNT_Type`
--

DROP TABLE IF EXISTS `UNT_Type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `UNT_Type` (
  `unt_type_id` int NOT NULL AUTO_INCREMENT,
  `first_type_id` int DEFAULT NULL,
  `second_type_id` int DEFAULT NULL,
  `second_subject_name` varchar(255) DEFAULT NULL,
  `second_subject_score` int DEFAULT '0',
  `history_of_kazakhstan` int DEFAULT '0',
  `mathematical_literacy` int DEFAULT '0',
  `reading_literacy` int DEFAULT '0',
  `TotalScore` int DEFAULT NULL,
  `history_of_kazakhstan_creative` int DEFAULT '0',
  `reading_literacy_creative` int DEFAULT '0',
  `total_score_creative` int DEFAULT '0',
  `total_score` int DEFAULT '0',
  `type` varchar(255) NOT NULL,
  `second_type_history_kazakhstan` int DEFAULT '0',
  `second_type_reading_literacy` int DEFAULT '0',
  `creative_exam1` int DEFAULT '0',
  `creative_exam2` int DEFAULT '0',
  `first_subject_score` int DEFAULT NULL,
  `school_id` int DEFAULT NULL,
  PRIMARY KEY (`unt_type_id`),
  KEY `first_type_id` (`first_type_id`),
  KEY `second_type_id` (`second_type_id`),
  CONSTRAINT `unt_type_ibfk_1` FOREIGN KEY (`first_type_id`) REFERENCES `First_Type` (`first_type_id`),
  CONSTRAINT `unt_type_ibfk_2` FOREIGN KEY (`second_type_id`) REFERENCES `Second_Type` (`second_type_id`)
) ENGINE=InnoDB AUTO_INCREMENT=41 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `UNT_Type`
--

LOCK TABLES `UNT_Type` WRITE;
/*!40000 ALTER TABLE `UNT_Type` DISABLE KEYS */;
INSERT INTO `UNT_Type` VALUES (3,1,7,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(4,9,3,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(5,9,5,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(6,9,8,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(8,1,8,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(9,19,4,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(10,26,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(11,27,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(12,27,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(13,27,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(14,NULL,9,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(15,27,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(16,28,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(17,28,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(18,NULL,9,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(19,29,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(20,30,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(21,35,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(22,NULL,14,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(23,NULL,14,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-2',0,0,0,0,NULL,NULL),(24,36,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(25,37,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(26,38,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(27,NULL,15,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-2',0,0,0,0,NULL,NULL),(28,39,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(29,40,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(30,41,NULL,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-1',0,0,0,0,NULL,NULL),(31,NULL,17,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-2',0,0,0,0,NULL,NULL),(32,NULL,18,NULL,0,0,0,0,NULL,NULL,NULL,0,0,'type-2',0,0,0,0,NULL,NULL),(33,NULL,18,NULL,0,0,0,0,NULL,0,0,0,NULL,'type-2',NULL,NULL,NULL,NULL,NULL,NULL),(34,30,NULL,NULL,0,0,0,0,NULL,0,0,NULL,0,'type-1',NULL,NULL,NULL,NULL,NULL,NULL),(35,30,NULL,NULL,0,0,0,0,NULL,0,0,NULL,0,'type-1',NULL,NULL,NULL,NULL,NULL,NULL),(36,48,NULL,NULL,0,0,0,0,NULL,0,0,NULL,0,'type-1',NULL,NULL,NULL,NULL,NULL,NULL),(37,49,NULL,NULL,0,0,0,0,NULL,0,0,NULL,0,'type-1',NULL,NULL,NULL,NULL,NULL,NULL),(38,50,NULL,NULL,0,0,0,0,NULL,0,0,NULL,0,'type-1',NULL,NULL,NULL,NULL,NULL,NULL),(39,51,NULL,NULL,0,0,0,0,NULL,0,0,NULL,0,'type-1',NULL,NULL,NULL,NULL,NULL,NULL),(40,55,NULL,NULL,0,0,0,0,NULL,0,0,NULL,0,'type-1',NULL,NULL,NULL,NULL,NULL,NULL);
/*!40000 ALTER TABLE `UNT_Type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `email` varchar(255) NOT NULL,
  `phone` varchar(20) DEFAULT NULL,
  `password` varchar(255) NOT NULL,
  `first_name` varchar(100) DEFAULT NULL,
  `last_name` varchar(100) DEFAULT NULL,
  `age` int DEFAULT NULL,
  `role` varchar(20) DEFAULT NULL,
  `verified` tinyint(1) DEFAULT '0',
  `otp_code` varchar(10) DEFAULT NULL,
  `is_active` tinyint(1) DEFAULT '0',
  `created_by` int DEFAULT NULL,
  `is_verified` tinyint(1) DEFAULT '0',
  `verification_token` varchar(255) DEFAULT NULL,
  `school_id` int DEFAULT NULL,
  `avatar_url` varchar(255) DEFAULT 'https://your-default-avatar-url.com',
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`),
  UNIQUE KEY `phone` (`phone`)
) ENGINE=InnoDB AUTO_INCREMENT=41 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES (3,'director@example.com',NULL,'2ded8b87d88604e76134f1c39a263c14d1073a02ec0dfb6582cfebb926293394','John','Doe',NULL,'user',0,NULL,1,1,1,NULL,NULL,'https://your-default-avatar-url.com'),(9,'erkinbnz@gmail.com',NULL,'$2a$10$CHzGEeCODkXmbfbZtwN0puc/lRCO0qDl.XK66Ogin4PogNq.4eh4S','Alibek','Murat',20,'user',0,'335018',0,NULL,0,NULL,NULL,'https://your-default-avatar-url.com'),(10,'erkinbz@gmail.com',NULL,'$2a$10$6Jp1cxhJkdsci2xV96u1P.LS1YSaWEo73D3m6yvGBbpjH047lcTa6','Alibek','Murat',20,'user',0,'2529',0,NULL,0,NULL,24,'https://your-default-avatar-url.com'),(11,'qwerty@gmail.com',NULL,'$2a$10$vSZGgUIKMlROnHF1ZRwaVe.6g/lAOaOhn0YW2QACJ.LiVndOuFmkK','Alinur','S',20,'user',0,'6435',0,NULL,0,NULL,NULL,'https://your-default-avatar-url.com'),(12,'qwety@gmail.com',NULL,'$2a$10$Hk58Ulj9IRzDdV3MlDIZ2eadIuQoeDD/yLJDriMoWkfFnDM0b0VbS','Alinur','S',20,'user',0,'7635',0,NULL,0,NULL,NULL,'https://your-default-avatar-url.com'),(13,'zhanibek@gmail.com',NULL,'$2a$10$NuX50g4U45iEHb9ATa0GDOTDli/AGp1E.gy7bQ4PYTKGXmsJFG/aq','Alinur','S',20,'user',0,NULL,0,NULL,0,'1a0d1a37-875f-499b-8c7a-fad0fd9a738c',NULL,'https://your-default-avatar-url.com'),(14,'muratalibek43@gmail.com',NULL,'$2a$10$f19LMQ/.0senNASqCFTGvuQxr0DEWi7uQQ1EWXZyO087iZ.D8yY8.','Alinur','S',20,'user',0,NULL,0,NULL,0,'a733b516-ca2a-4bf6-a901-413a43a68a60',NULL,'https://your-default-avatar-url.com'),(20,'alinur@gmail.com',NULL,'$2a$10$kA/Rj5gvKm7wUO021SZjuuRqX72AR7V3ThYBYvnbq1t5r1YzFbogS','Laura','S',20,'user',0,'8777',0,NULL,0,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFsaW51ckBnbWFpbC5jb20iLCJleHAiOjE3NDA0MTcwNTN9.nbL-q3LjxgGAe3mm_5Y_8RoIfckXX1QQDdshdnW1wGs',NULL,'https://your-default-avatar-url.com'),(22,'Alibek@gmail.com',NULL,'$2a$10$j1xq4.cfw3plc/NjunkhueLMJQbgkedPbOPuiw1opYq5ASpNEJSwa','Alinur','Alinur',20,'superadmin',0,'1870',0,NULL,0,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6IkFsaWJla0BnbWFpbC5jb20iLCJleHAiOjE3NDE3NTMwOTJ9.So5nNk9Y3WGpBQk6433TxBqEw71pniCToE5h0HSI77k',NULL,'https://your-default-avatar-url.com'),(25,'user@example.com',NULL,'$2a$10$o3vv0btuR3c9omha411.7OeLhIeVY4kHLy2VEKSgYtc/pkwodo78e','John','Doe',25,'user',0,'0050',0,NULL,0,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InVzZXJAZXhhbXBsZS5jb20iLCJleHAiOjE3NDE4Nzg3NDB9.GV7CXFXiOLkGS3x_ZdXlDgmhOHY2jzQY1Ff5KIuWxwo',NULL,'https://your-default-avatar-url.com'),(27,'erkinbrendkz@gmail.com',NULL,'$2a$10$m6Zd.TSURPiBcrW8oWTW6uCF8KFisotCV/ZTnuJFSiM9MJd7Fp562','John','Doe',25,'director',0,'8938',0,NULL,1,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImVya2luYnJlbmRrekBnbWFpbC5jb20iLCJleHAiOjE3NDIxNTA2Mjl9.FPFqcdgLj1XqERNQ-qg2T824Wy-1ggBV6SHOtpoOKFc',29,'https://your-default-avatar-url.com'),(28,'mralibemurat27@gmail.com',NULL,'$2a$10$MNBMJJ6AtNYnvlAZLbaPb.EHWjZOw447uWW63pJPRUWLTlGi56siy','Murat','Alibek',25,'director',0,'5297',0,NULL,1,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6Im1yYWxpYmVrbXVyYXQyN0BnbWFpbC5jb20iLCJleHAiOjE3NDI2NzgzMjV9.0sIOmw6wrjLwY5lmX09edxTbtWIJ0weAKFhCZhqKQwc',36,'https://your-default-avatar-url.com'),(29,'zhanibek460@gmail.com',NULL,'$2a$10$srm1K/wbmX8To.g9HOm2VuelJs4YOztXfNs3hZ.vZ0VNzegDDTu1e','Zhanibek','Muratov',25,'director',0,'1539',0,NULL,1,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InpoYW5pYmVrNDYwQGdtYWlsLmNvbSIsImV4cCI6MTc0Mjg0Njc3OX0.6knw14nvf4ymMSNZwCNDP_6voWVHmymskOEq5N9qRpA',35,'https://your-default-avatar-url.com'),(30,'alikhan2217se@gmail.com',NULL,'$2a$10$4CQjEmFR3p5KjE2L1ZCQy.1K4/SGSciSn9IeBvPpeME21J1rkbcuG','Zhanibek','Muratov',25,'user',0,'8669',0,NULL,1,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFsaWtoYW4yMjE3c2VAZ21haWwuY29tIiwiZXhwIjoxNzQzMTU1MjAyfQ.LWogV8aHsh7lwPbp3_i9tVR0GoB28ic_KI-jKxOb_aw',NULL,'https://your-default-avatar-url.com'),(31,'anarab902@gmail.com',NULL,'$2a$10$UKohh6oBzf/QqjAaLwRi0uKotTmT5WGiCKRX73dqiAwNg4g14FIBK','Laura','Laura',20,'superadmin',0,'9396',0,NULL,1,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFuYXJhYjkwMkBnbWFpbC5jb20iLCJleHAiOjE3NDM1Mjg3NjF9.ZkvdbJluH1UTiLs9R9lsdk8nNY6qNZWYgx0K8qBaNnw',39,'https://avatarschoolrank.s3.eu-north-1.amazonaws.com/avatar-31-1743739362.jpg'),(32,'221543@astanait.edu.kz',NULL,'$2a$10$8xuOwbNdIbKfBWdmRX05MOlpvP7pc9FOMCd/LPpQePzOzE1yRirZG','LLL','Laura',20,'user',0,'7379',0,NULL,0,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6IjIyMTU0M0Bhc3RhbmFpdC5lZHUua3oiLCJleHAiOjE3NDM1NDMzMTB9.s5ayPymIe_pv_RpDB98gWIex_f6dLsSIVGOI6RGS6PQ',NULL,'https://your-default-avatar-url.com'),(35,'mralibekmuratц27@gmail.com',NULL,'$2a$10$BxSY/UWJt.517JMNikEHueomxOLSJ9RzVPGhN0JK2Uz1w1meVztQW','Laura','Laura',20,'user',0,'1548',0,NULL,0,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6Im1yYWxpYmVrbXVyYXTRhjI3QGdtYWlsLmNvbSIsImV4cCI6MTc0MzU0NTM4OX0.2zBaeKs9_C_DaZhKpZlqo5LWO3gQ0NpCh--2957mYvc',NULL,'https://your-default-avatar-url.com'),(39,'mralibekmurat27@gmail.com',NULL,'$2a$10$Z6lPP.2N8h8xdeJJSWoC2uDK7v.SmO9lsXp5wcDT9IkixTQqqrXZ2','Laura','Laura',20,'schooladmin',0,'1787',0,NULL,1,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6Im1yYWxpYmVrbXVyYXQyN0BnbWFpbC5jb20iLCJleHAiOjE3NDM2MDg4NDR9.aJG2P11cBwfQaXPPVUSPeUFrfbnM9XGbZFfWPWR4NuE',41,'https://your-bucket-name.s3.amazonaws.com/default-avatar.jpg'),(40,'mraliberat27@gmail.com',NULL,'$2a$10$DrxK7mMBeOHZ/c0rrkGKtOopHBXC67zRew5n7qhLNwLd4tARIj73C','Laura','Laura',20,'user',0,'9589',0,NULL,0,'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6Im1yYWxpYmVyYXQyN0BnbWFpbC5jb20iLCJleHAiOjE3NDQxMTg3NDJ9.jZsPE649fXnINMc1whSmn_WkoC0MwoQvO3OnzyCjTME',NULL,'https://your-bucket-name.s3.amazonaws.com/default-avatar.jpg');
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-04-07 19:20:48
