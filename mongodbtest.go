package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/go-while/nntp-mongodb-storage"
	//"github.com/go-while/nntp-storage"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"time"
)

var (
	NUM_CPUS    int = 4
	lockparchan chan struct{}
	pardonechan chan struct{}
)

func main() {
	runtime.GOMAXPROCS(NUM_CPUS)
	// Define command-line flags
	var flagTestCase string
	var iflagNumIterations int
	var flagNumIterations uint64
	var flagRandomUpDN bool
	var flagTestPar int

	flag.StringVar(&flagTestCase, "test-case", "", "Test cases: delete|read|no-compression|gzip|zlib")
	flag.IntVar(&iflagNumIterations, "test-num", 0, "Test Num: Any number >= 0 you want")
	flag.IntVar(&flagTestPar, "test-par", 1, "Test Parallel: Any number >= 1 you want")
	flag.BoolVar(&flagRandomUpDN, "randomUpDN", false, "set flag '-randomUpDN' to test randomUpDN() function")
	flag.Parse()
	flagNumIterations = uint64(iflagNumIterations)
	log.Printf("flagNumIterations=%d", flagNumIterations)
	lockparchan = make(chan struct{}, flagTestPar)
	pardonechan = make(chan struct{}, flagTestPar)

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
	use_format := "wireformat" // or: fileformat
	testCases := [][]string{}
	testRun00 := []string{"read", "delete", "no-compression", "read", "delete", "read"}
	testRun01 := []string{"read", "delete", "gzip", "read", "delete", "read"}
	testRun02 := []string{"read", "delete", "zlib", "read", "delete", "read"}
	testChaos := []string{"read", "delete", "zlib", "gzip", "no-compression"}
	//testRun1 := []string{"delete", "no-compression", "read", "delete", "gzip", "read", "delete", "zlib", "read"}
	//testRun2 := []string{"delete", "read", "delete", "read"}
	//testRun3 := []string{"delete", "read", "no-compression", "read", "read", "read"}
	//testRun4 := []string{"read", "delete", "no-compression", "read", "read", "delete", "read"}
	//testCases = append(testCases, testRun00)
	//testCases = append(testCases, testRun01)
	//testCases = append(testCases, testRun02)
	testCases = append(testCases, testRun00, testRun01, testRun02)
	//testCases = append(testCases, testRun00, testRun01, testRun02, testRun1, testRun2, testRun3, testRun4)
	if validTestCase && testCase != "" {
		log.Printf("Running '%s' test case...", testCase)
		testCases = [][]string{}
		testCases = append(testCases, []string{testCase})
	}
	TESTMongoDatabaseName := "nntp_TEST" // mongosh: use nntp_TEST; db.articles.drop();

	// load mongodb storage
	cfg := mongostorage.GetDefaultMongoStorageConfig()

	cfg.DelQueue = 4
	cfg.DelWorker = 4
	cfg.DelBatch = 10000

	cfg.InsQueue = 4
	cfg.InsWorker = 4
	cfg.InsBatch = 10000
	cfg.TestAfterInsert = false

	cfg.GetQueue = 4
	cfg.GetWorker = 4

	cfg.FlushTimer = 2500

	cfg.MongoURI = ""
	cfg.MongoDatabaseName = TESTMongoDatabaseName
	//cfg.MongoTimeout := int64(86400)

	mongostorage.Load_MongoDB(&cfg)

	if flagRandomUpDN {
		go mongostorage.MongoWorker_UpDn_Random()
	}

	target := uint64(0)
	if flagTestPar > 0 {
		for i:=1; i <= flagTestPar; i++ {
			testCases = nil
			RandomStringSlice(testChaos)
			testCases = append(testCases, testChaos, testChaos, testChaos)
			RandomStringSlices(testCases)
			log.Printf("run %d/%d testCases='%v'", i, flagTestPar, testCases)
			for _, testRun := range testCases {
				for _, caseToTest := range testRun {
					if flagNumIterations > 0 {
						target += flagNumIterations
						go TestArticles(flagNumIterations, caseToTest, use_format, cfg.TestAfterInsert)
					}
				} // end for
			} // end for
		} // end for
	} else {
		for _, testRun := range testCases {
			for _, caseToTest := range testRun {
				if flagNumIterations > 0 {
					target += flagNumIterations
					TestArticles(flagNumIterations, caseToTest, use_format, cfg.TestAfterInsert)
				}
			} //emd for
		} // end for
	} // end if flagTestPar

wait:
	for {
		time.Sleep(time.Second)
		if len(pardonechan) == flagTestPar {
			r := mongostorage.Counter.Get("Did_mongoWorker_Reader")
			i := mongostorage.Counter.Get("Did_mongoWorker_Insert")
			d := mongostorage.Counter.Get("Did_mongoWorker_Delete")
			sum := r + i + d
			if sum == target {
				log.Printf("Test completed: %d/%d r=%d i=%d d=%d target=%d", len(pardonechan), flagTestPar, r, i, d, target)
				break wait
			}
			log.Printf("Waiting for Test to complete: %d/%d r=%d i=%d d=%d target=%d/%d", len(pardonechan), flagTestPar, r, i, d, sum, target)
			continue wait
		}
		log.Printf("Waiting for Test to complete: %d/%d", len(pardonechan), flagTestPar)
	} // end for

	log.Printf("Closing mongodbtest")
	time.Sleep(time.Second)
	close(mongostorage.Mongo_Delete_queue)
	close(mongostorage.Mongo_Insert_queue)
	close(mongostorage.Mongo_Reader_queue)
	time.Sleep(time.Second * 5)
	log.Printf("Quit mongodbtest")
} // end func main

// function written by AI.
func hashMessageID(messageID string) string {
	hash := sha256.New()
	hash.Write([]byte(messageID))
	return hex.EncodeToString(hash.Sum(nil))
} // end func hashMessageID

func LockPar(caseToTest *string) {
forever:
	for {
		select {
			case lockparchan <- struct{}{}:
				//log.Printf("OK LockPar %s", *caseToTest)
				break forever
			default:
				time.Sleep(time.Second)
				//log.Printf("Wait LockPar %s", *caseToTest)
		}
	}
} // end func LockPar

func UnLockPar(caseToTest *string) {
	<-lockparchan
	pardonechan <- struct{}{}
	//log.Printf("UnLockPar %s", *caseToTest)
} //end func UnLockPar

func RandomStringSlice(slice []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
} // end func RandomStringSlice


func RandomStringSlices(slices [][]string) {
	for _, slice := range slices {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(slice), func(i, j int) {
			slice[i], slice[j] = slice[j], slice[i]
		})
	}
} // end func RandomStringSlices

// TestArticles is responsible for inserting articles into the MongoDB collection
// based on the specified test case. It generates example articles and performs different
// actions based on the 'caseToTest' parameter, which determines the compression method
// used for storing articles or deletes existing articles. The function also checks for
// existing articles with the same MessageIDHash before inserting new articles to avoid
// duplicates. It runs multiple iterations for the given number of inserts and provides
// logging information for each step of the process.
//
// The 'caseToTest' parameter determines the behavior of the function as follows:
// - "read": Articles are read from the MongoDB collection based on the given MessageIDHash.
// - "no-compression": Articles are inserted without any compression.
// - "gzip": Articles are compressed using gzip before insertion.
// - "zlib": Articles are compressed using zlib before insertion.
// - "delete": Existing articles with the given MessageIDHash are deleted.
//
// For each test case, the function logs details such as the raw size of the article,
// the size after compression (if applicable), and whether the article has been inserted
// successfully. It also handles deleting articles based on the 'delete' case, and
// reports any errors that occur during the insertion or deletion process.
// function partly written by AI.
func TestArticles(NumIterations uint64, caseToTest string, use_format string, checkAfterInsert bool) {
	time.Sleep(time.Second)
	LockPar(&caseToTest)
	defer UnLockPar(&caseToTest)
	log.Printf("run test: case '%s'", caseToTest)
	defer log.Printf("end test: case '%s'", caseToTest)

	insreqs, delreqs, readreqs := 0, 0, 0
	t_get, t_ins, t_del, t_nf := 0, 0, 0, 0

	for i := uint64(1); i <= NumIterations; i++ {

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
			MessageIDHash: &messageIDHash,
			MessageID:     &messageID,
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
			t_del++
			delreqs++
			if delreqs >= 1000 {
				delreqs = 0
				log.Printf("Add #%d/%d to Mongo_Delete_queue=%d/%d", t_del, NumIterations, len(mongostorage.Mongo_Delete_queue), cap(mongostorage.Mongo_Delete_queue))
			}
			mongostorage.Mongo_Delete_queue <- messageIDHash

		case "read":
			readreqs++
			//log.Printf("Add #%d to Mongo_Reader_queue=%d/%d", readreqs, len(mongostorage.Mongo_Reader_queue), cap(mongostorage.Mongo_Reader_queue))
			// test additional not existant hash with this read request
			//hash2 := "none2"
			//hash3 := "none3"
			// retchan is a buffered channel with a capacity of 1 to store slices of pointers to Mongostorage.MongoArticle objects.
			// It is used to send the retrieved articles back to the caller when performing a read operation.
			retchan := make(chan []*mongostorage.MongoArticle, 1)
			readreq := mongostorage.MongoReadRequest{
				//Msgidhashes: []string{messageIDHash, hash2, hash3},
				Msgidhashes: []*string{&messageIDHash},
				RetChan:     retchan,
			}
			mongostorage.Mongo_Reader_queue <- readreq
			//log.Printf("read waiting for reply on RetChan q=%d", len(mongostorage.Mongo_Reader_queue))
			//timeout := time.After(time.Duration(mongostorage.DefaultMongoTimeout + 1))
			select {
			//case <-timeout:
			//	log.Printf("Error readreq.retchan request timed out")
			//	break
			case articles, ok := <-retchan:
				if !ok {
					log.Printf("INFO readreq.retchan has been closed")
				}
				if len(articles) == 0 {
					log.Printf("Error testCase: 'read' empty reply from retchan hash='%s'", *article.MessageIDHash)
				} else {
					got_read := 0
					not_found := 0
					for _, article := range articles {
						if article.Found {
							switch article.Enc {
							case mongostorage.NOCOMP:
								// pass
							case mongostorage.GZIP_enc:
								if uncompressedHead, err := mongostorage.DecompressData([]byte(article.Head), mongostorage.GZIP_enc); err != nil {
									log.Printf("Error GzipUncompress Head returned err: %v", err)
									continue
								} else {
									//log.Printf("OK GzipUncompressed Head")
									article.Head = uncompressedHead
								}

								if uncompressedBody, err := mongostorage.DecompressData([]byte(article.Body), mongostorage.GZIP_enc); err != nil {
									log.Printf("Error GzipUncompress Body returned err: %v", err)
									continue
								} else {
									//log.Printf("OK GzipUncompressed Body")
									article.Body = uncompressedBody
								}

							case mongostorage.ZLIB_enc:
								if uncompressedHead, err := mongostorage.DecompressData([]byte(article.Head), mongostorage.ZLIB_enc); err != nil {
									log.Printf("Error ZlibUncompress Head returned err: %v", err)
									continue
								} else {
									//log.Printf("OK ZlibUncompressed Head")
									article.Head = uncompressedHead
								}

								if uncompressedBody, err := mongostorage.DecompressData([]byte(article.Body), mongostorage.ZLIB_enc); err != nil {
									log.Printf("Error ZlibUncompress Body returned err: %v", err)
									continue
								} else {
									//log.Printf("OK ZlibUncompressed Body")
									article.Body = uncompressedBody
								}

							} // end switch encoder

							got_read++
							t_get++
							//log.Printf("testCase: 'read' got_read=%d/%d head='%s' body='%s' msgid='%s' hash=%s a.found=%t enc=%d", got_read, len(articles), string(article.Head), string(article.Body), *article.MessageID, *article.MessageIDHash, article.Found, article.Enc)
							//
							if got_read >= 1000 {
								log.Printf("testCase: 'read' got_read=%d/%d", t_get, len(articles), *article.MessageIDHash, article.Found, article.Enc)
								got_read = 0
							}
						} else {
							t_nf++
							not_found++
							if not_found >= 1000 {
								not_found = 0
								log.Printf("testCase: 'read' not_found=%d/%d hash=%s a.found=%t", t_nf, len(articles), *article.MessageIDHash, article.Found)
							}
						}
					} // for article := range articles
				} // end if len(articles)
			} // end select

		case "no-compression":
			// Inserts the article into MongoDB without compression
			insreqs++
			t_ins++
			//log.Printf("Add #%d to Mongo_Insert_queue=%d/%d", insreqs, len(mongostorage.Mongo_Insert_queue), cap(mongostorage.Mongo_Insert_queue))
			//log.Printf("insert (caseToTest=%s): %s", caseToTest, article.MessageIDHash)
			article.Headsize = len(article.Head)
			article.Bodysize = len(article.Body)
			article.Enc = mongostorage.NOCOMP
			mongostorage.Mongo_Insert_queue <- article

		case "gzip":
			// Inserts the article into MongoDB with gzip compression
			insreqs++
			t_ins++
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
			//log.Printf("Head raw=%d gzip=%d Body raw=%d gzip=%d msghash=%s", len(article.Head), len(compressedHead), len(article.Body), len(compressedBody), *article.MessageIDHash)
			//log.Printf("", )
			article.Head = compressedHead
			article.Headsize = len(compressedHead)
			article.Body = compressedBody
			article.Bodysize = len(compressedBody)
			article.Enc = mongostorage.GZIP_enc
			//log.Printf("insert (caseToTest=%s): %s", caseToTest, )
			mongostorage.Mongo_Insert_queue <- article
			//mongostorage.Mongo_Delete_queue <- article.MessageIDHash

		case "zlib":
			// Inserts the article into MongoDB with zlib compression
			insreqs++
			t_ins++
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
			//log.Printf("Head raw=%d gzip=%d Body raw=%d gzip=%d msghash=%s", len(article.Head), len(compressedHead), len(article.Body), len(compressedBody), *article.MessageIDHash)
			article.Head = compressedHead
			article.Headsize = len(compressedHead)
			article.Body = compressedBody
			article.Bodysize = len(compressedBody)
			article.Enc = mongostorage.ZLIB_enc
			//log.Printf("insert (caseToTest=%s): %s", caseToTest, *article.MessageIDHash)
			mongostorage.Mongo_Insert_queue <- article
			//mongostorage.Mongo_Delete_queue <- article.MessageIDHash

		default:
			log.Fatalf("Invalid case to test: %s", caseToTest)
		} // ens switch caseToTest
	}
} // end func TestArticles

// EOF mongodbtest.go
