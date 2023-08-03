package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/go-while/nntp-mongodb-storage"
	//"github.com/go-while/nntp-storage"
	"log"
	"strings"
	"time"
)

var (
	numInserts int = 5000000
)

func main() {
	use_format := "wireformat" // or: fileformat
	//testCases := []string{"delete", "no-compression", "gzip-only", "zlib-only"}
	testCases := []string{"no-compression", "delete"}
	checkAfterInsert := false
	TESTmongoDatabaseName := "nntp_TEST"
	// load mongodb storage
	delQueue := mongostorage.DefaultDelQueue
	delWorker := mongostorage.DefaultDelWorker
	insQueue := mongostorage.DefaultInsQueue
	insWorker := mongostorage.DefaultInsWorker
	testAfterInsert := false
	mongostorage.Load_MongoDB(mongostorage.DefaultMongoUri, TESTmongoDatabaseName, mongostorage.DefaultMongoCollection, mongostorage.DefaultMongoTimeout, delWorker, delQueue, insWorker, insQueue, testAfterInsert)

	for _, caseToTest := range testCases {
		log.Printf("run case: %s", caseToTest)
		TestInsertArticles(numInserts, caseToTest, use_format, checkAfterInsert)
	}
	close(mongostorage.Mongo_Delete_queue)
	close(mongostorage.Mongo_Insert_queue)

	time.Sleep(time.Second * 9)
} // end func main

func hashMessageID(messageID string) string {
	hash := sha256.New()
	hash.Write([]byte(messageID))
	return hex.EncodeToString(hash.Sum(nil))
} // end func hashMessageID

// TestInsertArticles is responsible for inserting articles into the MongoDB collection
// based on the specified test case. It generates example articles and performs different
// actions based on the 'caseToTest' parameter, which determines the compression method
// used for storing articles or deletes existing articles. The function also checks for
// existing articles with the same MessageIDHash before inserting new articles to avoid
// duplicates. It runs multiple iterations for the given number of inserts and provides
// logging information for each step of the process.
//
// The 'caseToTest' parameter determines the behavior of the function as follows:
// - "no-compression": Articles are inserted without any compression.
// - "gzip-only": Articles are compressed using gzip before insertion.
// - "zlib-only": Articles are compressed using zlib before insertion.
// - "delete": Existing articles with the given MessageIDHash are deleted.
//
// For each test case, the function logs details such as the raw size of the article,
// the size after compression (if applicable), and whether the article has been inserted
// successfully. It also handles deleting articles based on the 'delete' case, and
// reports any errors that occur during the insertion or deletion process.
func TestInsertArticles(numInserts int, caseToTest string, use_format string, checkAfterInsert bool) {

	for i := 0; i < numInserts; i++ {

		// Example data to store in the Article.
		messageID := fmt.Sprintf("<example-%d@example.com>", i)
		messageIDHash := hashMessageID(messageID)

		headerLines := []string{
			"From: John Doe <johndoe@example.com>",
			"Newsgroups: golang, misc.test",
			"Subject: Example Usenet Article",
			"Date: Thu, 02 Aug 2023 12:34:56 -0700",
			"Message-ID: " + messageID,
		}

		bodyLines := []string{
			"This is an example article.",
			"It has multiple lines.",
			"3rd line.",
			"4th line.",
			"5th line.",
		}

		// Create the Article.
		article := mongostorage.MongoArticle{
			MessageIDHash: messageIDHash,
			MessageID:     messageID,
		}
		switch use_format {
		case "wireformat":
			article.Head = []byte(strings.Join(headerLines, "\r\n")) // wireformat
			article.Body = []byte(strings.Join(bodyLines, "\r\n"))   // wireformat
		case "fileformat":
			article.Head = []byte(strings.Join(headerLines, "\n")) // fileformat
			article.Body = []byte(strings.Join(bodyLines, "\n"))   // fileformat
		}
		if len(article.Head) == 0 || len(article.Body) == 0 {
			log.Printf("IGNORE empty head=%d body=%d ", len(article.Head), len(article.Body))
			continue
		}

		switch caseToTest {
		case "delete":
			//log.Printf("add to Mongo_Delete_queue=%d/%d", len(mongostorage.Mongo_Delete_queue), cap(mongostorage.Mongo_Delete_queue))
			mongostorage.Mongo_Delete_queue <- messageIDHash

		case "no-compression":
			// Inserts the article into MongoDB without compression
			//log.Printf("insert (caseToTest=%s): %s", caseToTest, article.MessageIDHash)
			mongostorage.Mongo_Insert_queue <- article

		case "gzip-only":
			// Inserts the article into MongoDB with gzip compression
			var err error
			var compressedHead []byte
			var compressedBody []byte
			if compressedHead, err = mongostorage.GzipCompress([]byte(article.Head)); err != nil {
				log.Printf("Error gzipCompress Head returned err: %v", err)
				continue
			}
			if compressedBody, err = mongostorage.GzipCompress([]byte(article.Body)); err != nil {
				log.Printf("Error gzipCompress Body returned err: %v", err)
				continue
			}
			log.Printf("Head raw=%d gzip=%d | Body raw=%d gzip=%d", len(article.Head), len(compressedHead), len(article.Body), len(compressedBody))
			article.Head = compressedHead
			article.Headsize = len(compressedHead)
			article.Body = compressedBody
			article.Bodysize = len(compressedBody)
			//log.Printf("insert (caseToTest=%s): %s", caseToTest, article.MessageIDHash)
			mongostorage.Mongo_Insert_queue <- article
			//mongostorage.Mongo_Delete_queue <- article.MessageIDHash

		case "zlib-only":
			// Inserts the article into MongoDB with zlib compression
			var err error
			var compressedHead []byte
			var compressedBody []byte
			if compressedHead, err = mongostorage.ZlibCompress([]byte(article.Head)); err != nil {
				log.Printf("Error zlibCompress Head returned err='%v'", err)
				continue
			}
			if compressedBody, err = mongostorage.ZlibCompress([]byte(article.Body)); err != nil {
				log.Printf("Error zlibCompress Body returned err='%v'", err)
				continue
			}
			log.Printf("Head raw=%d zlib=%d", len(article.Head), len(compressedHead))
			log.Printf("Body raw=%d zlib=%d", len(article.Body), len(compressedBody))
			article.Head = compressedHead
			article.Headsize = len(compressedHead)
			article.Body = compressedBody
			article.Bodysize = len(compressedBody)
			//log.Printf("insert (caseToTest=%s): %s", caseToTest, article.MessageIDHash)
			mongostorage.Mongo_Insert_queue <- article
			//mongostorage.Mongo_Delete_queue <- article.MessageIDHash

		default:
			log.Fatalf("Invalid case to test: %s", caseToTest)
		} // ens switch caseToTest
	}
} // end func TestInsertArticles

// EOF mongodbtest.go
