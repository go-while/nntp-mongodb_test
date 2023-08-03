package main

import (
    "crypto/sha256"
    "encoding/hex"
    "flag"
    "fmt"
    "github.com/go-while/nntp-mongodb-storage"
    //"github.com/go-while/nntp-storage"
    "log"
    "runtime"
    "strings"
    "time"
)

var (
    NUM_CPUS   int = 4
    NumIterations int = 10
)

func main() {
    runtime.GOMAXPROCS(NUM_CPUS)
    // Define command-line flags
    var flagTestCase string
    var flagNumIterations int

    flag.StringVar(&flagTestCase, "test-case", "", "Test cases: delete|read|no-compression|gzip|zlib")
    flag.IntVar(&flagNumIterations, "test-num", 0, "Test Num: Any number you want")

    flag.Parse()

    // Define supported test cases
    supportedTestCases := []string{"delete", "read", "no-compression", "gzip", "zlib"}
    testCase := ""
    validTestCase := false
    for _, tc := range supportedTestCases {
        if flagTestCase == tc {
            testCase = tc
            validTestCase = true
            break
        }
    }
    if flagNumIterations > 0 {
        NumIterations = flagNumIterations
    }
    use_format := "wireformat" // or: fileformat
    testCases := [][]string{}
    //testRun00 := []string{"delete", "gzip", "read", "delete"}
    //testRun01 := []string{"delete", "zlib", "read", "delete"}
    testRun02 := []string{"delete", "no-compression", "read", "delete"}
    //testRun1 := []string{"delete", "no-compression", "read", "delete", "gzip", "read", "delete", "zlib", "read"}
    //testRun2 := []string{"delete", "read", "delete", "read"}
    //testRun3 := []string{"delete", "read", "no-compression", "read", "read", "read"}
    //testRun4 := []string{"read", "delete", "no-compression", "read", "read", "delete", "read"}
    testCases = append(testCases, testRun02)
    //testCases = append(testCases, testRun00, testRun01, testRun02)
    //testCases = append(testCases, testRun00, testRun01, testRun02, testRun1, testRun2, testRun3, testRun4)
    if validTestCase && testCase != "" {
        log.Printf("Running '%s' test case...", testCase)
        testCases = [][]string{}
        testCases = append(testCases, []string{testCase})
    }
    checkAfterInsert := false
    TESTmongoDatabaseName := "nntp_TEST" // mongosh: use nntp_TEST; db.articles.drop();
    // load mongodb storage
    delQueue := mongostorage.DefaultDelQueue
    delWorker := mongostorage.DefaultDelWorker
    insQueue := mongostorage.DefaultInsQueue
    insWorker := mongostorage.DefaultInsWorker
    getQueue := mongostorage.DefaultGetQueue
    getWorker := mongostorage.DefaultGetWorker
    mongoTimeout := mongostorage.DefaultMongoTimeout
    //mongoTimeout := int64(86400)
    testAfterInsert := false
    mongostorage.Load_MongoDB(mongostorage.DefaultMongoUri, TESTmongoDatabaseName, mongostorage.DefaultMongoCollection, mongoTimeout, delWorker, delQueue, insWorker, insQueue, getQueue, getWorker, testAfterInsert)

    c := 0
    for _, testRun := range testCases {
        for _, caseToTest := range testRun {
            c++
            log.Printf("run test %d/%d: case '%s'", c, len(testCases), caseToTest)
            TestArticles(NumIterations, caseToTest, use_format, checkAfterInsert)
            log.Printf("end test %d/%d: case '%s'", c, len(testCases), caseToTest)
            time.Sleep(time.Second * 5)
        }
    }
    time.Sleep(time.Second * 5)
    log.Printf("Closing mongodbtest")
    close(mongostorage.Mongo_Delete_queue)
    close(mongostorage.Mongo_Insert_queue)
    close(mongostorage.Mongo_Reader_queue)
    time.Sleep(time.Second * 5)
    log.Printf("Quit mongodbtest")
} // end func main

func hashMessageID(messageID string) string {
    hash := sha256.New()
    hash.Write([]byte(messageID))
    return hex.EncodeToString(hash.Sum(nil))
} // end func hashMessageID

// TestArticles is responsible for inserting articles into the MongoDB collection
// based on the specified test case. It generates example articles and performs different
// actions based on the 'caseToTest' parameter, which determines the compression method
// used for storing articles or deletes existing articles. The function also checks for
// existing articles with the same MessageIDHash before inserting new articles to avoid
// duplicates. It runs multiple iterations for the given number of inserts and provides
// logging information for each step of the process.
//
// The 'caseToTest' parameter determines the behavior of the function as follows:
// - "no-compression": Articles are inserted without any compression.
// - "gzip": Articles are compressed using gzip before insertion.
// - "zlib": Articles are compressed using zlib before insertion.
// - "delete": Existing articles with the given MessageIDHash are deleted.
//
// For each test case, the function logs details such as the raw size of the article,
// the size after compression (if applicable), and whether the article has been inserted
// successfully. It also handles deleting articles based on the 'delete' case, and
// reports any errors that occur during the insertion or deletion process.
func TestArticles(NumIterations int, caseToTest string, use_format string, checkAfterInsert bool) {
    insreqs, delreqs, readreqs := 0, 0, 0
    for i := 1; i <= NumIterations; i++ {

        // Example data to store in the Article.
        messageID := fmt.Sprintf("<example-%d@example.com>", i)
        messageIDHash := hashMessageID(messageID)

        headerLines := []string{
            "From: John Doe <johndoe@example.com>",
            "Newsgroups: golang, misc.test",
            "Subject: Example Usenet Article",
            "Date: Thu, 02 Aug 2023 12:34:56 -0700",
            //fmt.Sprintf("X-MSGID-SHA256: %s", messageIDHash),
            fmt.Sprintf("Message-ID: %s", messageID),
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
            delreqs++
            log.Printf("add #%d to Mongo_Delete_queue=%d/%d", delreqs, len(mongostorage.Mongo_Delete_queue), cap(mongostorage.Mongo_Delete_queue))
            mongostorage.Mongo_Delete_queue <- messageIDHash

        case "read":
            readreqs++
            log.Printf("add #%d to Mongo_Reader_queue=%d/%d", readreqs, len(mongostorage.Mongo_Reader_queue), cap(mongostorage.Mongo_Reader_queue))
            // test additional not existant hash with this read request
            hash2 := "none2"
            hash3 := "none3"
            retchan := make(chan []*mongostorage.MongoArticle, 1)
            readreq := mongostorage.MongoReadRequest{
                Msgidhashes: []string{messageIDHash, hash2, hash3},
                RetChan:     retchan,
            }
            mongostorage.Mongo_Reader_queue <- readreq
            //timeout := time.After(time.Duration(mongostorage.DefaultMongoTimeout*2))
            select {
            case articles, ok := <-retchan:
                if !ok {
                    log.Printf("readreq.retchan has been closed")
                }
                if len(articles) == 0 {
                    log.Printf("Error testCase: 'read' empty reply from retchan hash='%s'", article.MessageIDHash)
                } else {
                    got_read := 0
                    not_found := 0
                    for _, article := range articles {
                        if article.Found {
                            switch(article.Enc) {
                            case mongostorage.NOCOMP:
                                // pass
                            case mongostorage.GZIP_enc:
                                if uncompressedHead, err := mongostorage.DecompressData([]byte(article.Head), mongostorage.GZIP_enc); err != nil {
                                    log.Printf("Error GzipUncompress Head returned err: %v", err)
                                    continue
                                } else {
                                    article.Head = uncompressedHead
                                }

                                if uncompressedBody, err := mongostorage.DecompressData([]byte(article.Body), mongostorage.GZIP_enc); err != nil {
                                    log.Printf("Error GzipUncompress Body returned err: %v", err)
                                    continue
                                } else {
                                    article.Body = uncompressedBody
                                }

                            case mongostorage.ZLIB_enc:
                                if uncompressedHead, err := mongostorage.DecompressData([]byte(article.Head), mongostorage.ZLIB_enc); err != nil {
                                    log.Printf("Error ZlibUncompress Head returned err: %v", err)
                                    continue
                                } else {
                                    article.Head = uncompressedHead
                                }

                                if uncompressedBody, err := mongostorage.DecompressData([]byte(article.Body), mongostorage.ZLIB_enc); err != nil {
                                    log.Printf("Error ZlibUncompress Body returned err: %v", err)
                                    continue
                                } else {
                                    article.Body = uncompressedBody
                                }

                            } // end switch encoder

                            got_read++
                            log.Printf("testCase: 'read' got_read=%d/%d head='%s' body='%s' msgid='%s' hash=%s a.found=%t enc=%d", got_read, len(articles), string(article.Head), string(article.Body), article.MessageID, article.MessageIDHash, article.Found, article.Enc)
                        } else {
                            not_found++
                            log.Printf("testCase: 'read' not_found=%d/%d hash=%s a.found=%t", not_found, len(articles), article.MessageIDHash, article.Found)
                        }
                    }
                }
                /*
                   case <- timeout:
                       log.Printf("readreq.retchan request timed out")

                       break
                */
            } // end select
            //timeout = nil

        case "no-compression":
            // Inserts the article into MongoDB without compression
            insreqs++
            log.Printf("add #%d to Mongo_Insert_queue=%d/%d", insreqs, len(mongostorage.Mongo_Insert_queue), cap(mongostorage.Mongo_Insert_queue))
            //log.Printf("insert (caseToTest=%s): %s", caseToTest, article.MessageIDHash)
            article.Enc = mongostorage.NOCOMP
            mongostorage.Mongo_Insert_queue <- article

        case "gzip":
            // Inserts the article into MongoDB with gzip compression
            var err error
            var compressedHead []byte
            var compressedBody []byte
            if compressedHead, err = mongostorage.CompressData([]byte(article.Head), mongostorage.GZIP_enc); err != nil {
                log.Printf("Error GzipCompress Head returned err: %v", err)
                continue
            }
            if compressedBody, err = mongostorage.CompressData([]byte(article.Body), mongostorage.GZIP_enc); err != nil {
                log.Printf("Error GzipCompress Body returned err: %v", err)
                continue
            }
            log.Printf("Head raw=%d gzip=%d | Body raw=%d gzip=%d", len(article.Head), len(compressedHead), len(article.Body), len(compressedBody))
            article.Head = compressedHead
            article.Headsize = len(compressedHead)
            article.Body = compressedBody
            article.Bodysize = len(compressedBody)
            article.Enc = mongostorage.GZIP_enc
            //log.Printf("insert (caseToTest=%s): %s", caseToTest, article.MessageIDHash)
            mongostorage.Mongo_Insert_queue <- article
            //mongostorage.Mongo_Delete_queue <- article.MessageIDHash

        case "zlib":
            // Inserts the article into MongoDB with zlib compression
            var err error
            var compressedHead []byte
            var compressedBody []byte
            if compressedHead, err = mongostorage.CompressData([]byte(article.Head), mongostorage.ZLIB_enc); err != nil {
                log.Printf("Error ZlibCompress Head returned err='%v'", err)
                continue
            }
            if compressedBody, err = mongostorage.CompressData([]byte(article.Body), mongostorage.ZLIB_enc); err != nil {
                log.Printf("Error ZlibCompress Body returned err='%v'", err)
                continue
            }
            log.Printf("Head raw=%d zlib=%d", len(article.Head), len(compressedHead))
            log.Printf("Body raw=%d zlib=%d", len(article.Body), len(compressedBody))
            article.Head = compressedHead
            article.Headsize = len(compressedHead)
            article.Body = compressedBody
            article.Bodysize = len(compressedBody)
            article.Enc = mongostorage.ZLIB_enc
            //log.Printf("insert (caseToTest=%s): %s", caseToTest, article.MessageIDHash)
            mongostorage.Mongo_Insert_queue <- article
            //mongostorage.Mongo_Delete_queue <- article.MessageIDHash

        default:
            log.Fatalf("Invalid case to test: %s", caseToTest)
        } // ens switch caseToTest
    }
} // end func TestArticles

// EOF mongodbtest.go
