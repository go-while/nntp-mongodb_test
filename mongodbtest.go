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
	"time"
)

const (
	CR   string = mongostorage.CR
	LF   string = mongostorage.LF
	CRLF string = mongostorage.CRLF
)

var (
	DEBUG       bool
	lockparchan chan struct{}
	pardonechan chan struct{}
)

func main() {

	// Define command-line flags
	var flagTestCase string
	var iflagNumIterations int
	var flagNumIterations uint64
	var flagRandomUpDN bool
	var flagTestPar int
	var flagNumCPU int
	var flagFormat string

	flag.StringVar(&flagTestCase, "test-case", "", "Test cases: delete|read|no-compression|gzip|zlib")
	flag.StringVar(&flagFormat, "format", "wireformat", "[wireformat|fileformat]")
	flag.IntVar(&iflagNumIterations, "test-num", 10, "Test Num: Any number >= 0 you want")
	flag.IntVar(&flagTestPar, "test-par", 1, "Test Parallel: Any number > 0 you want")
	flag.IntVar(&flagNumCPU, "num-cpu", 0, "sets runtime.GOMAXPROCS")
	flag.BoolVar(&flagRandomUpDN, "randomUpDN", false, "set flag '-randomUpDN' to test randomUpDN() function")
	flag.BoolVar(&DEBUG, "debug", false, "set flag '-debug' to enable 'logf() debugs")
	flag.Parse()
	mongostorage.DEBUG = DEBUG
	if flagNumCPU > 0 {
		runtime.GOMAXPROCS(flagNumCPU)
	}
	flagNumIterations = uint64(iflagNumIterations)
	log.Printf("flagNumIterations=%d flagTestPar=%d", flagNumIterations, flagTestPar)
	lockparchan = make(chan struct{}, flagTestPar)

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
	testCases := [][]string{}
	testRun00 := []string{"read", "delete", "no-compression", "read", "delete", "read"}
	testRun01 := []string{"read", "delete", "gzip", "read", "delete", "read"}
	testRun02 := []string{"read", "delete", "zlib", "read", "delete", "read"}
	testChaos := []string{"read", "delete", "zlib", "gzip", "no-compression", "delete", "delete", "delete"}
	//testRun1 := []string{"delete", "no-compression", "read", "delete", "gzip", "read", "delete", "zlib", "read"}
	//testRun2 := []string{"delete", "read", "delete", "read"}
	//testRun3 := []string{"delete", "read", "no-compression", "read", "read", "read"}
	//testRun4 := []string{"read", "delete", "no-compression", "read", "read", "delete", "read"}
	//testCases = append(testCases, testRun00)
	//testCases = append(testCases, testRun01)
	//testCases = append(testCases, testRun02)
	testCases = append(testCases, testRun00, testRun01, testRun02)
	//testCases = append(testCases, testRun00, testRun01, testRun02, testRun1, testRun2, testRun3, testRun4)

	// capture the flag
	if validTestCase && testCase != "" {
		log.Printf("Running '%s' test case...", testCase)
		testCases = [][]string{}
		testCases = append(testCases, []string{testCase})

	} else {
		log.Printf("Running '%s' test case...", testCases)
	}

	lockparchansize := 0
	for _, cases := range testCases {
		for _, _ = range cases {
			lockparchansize++
		}
	}
	log.Printf("Jobs=%d", lockparchansize)
	pardonechan = make(chan struct{}, lockparchansize)
	TESTMongoDatabaseName := "nntp_TEST" // mongosh: use nntp_TEST; db.articles.drop();

	// load mongodb storage
	cfg := mongostorage.GetDefaultMongoStorageConfig()

	cfg.DelQueue = 10
	cfg.DelWorker = 1
	cfg.DelBatch = 2

	cfg.InsQueue = 10
	cfg.InsWorker = 1
	cfg.InsBatch = 100
	cfg.TestAfterInsert = false

	cfg.GetQueue = 10
	cfg.GetWorker = 1

	cfg.FlushTimer = 1000

	cfg.MongoURI = ""
	cfg.MongoDatabaseName = TESTMongoDatabaseName
	//cfg.MongoTimeout := int64(86400)

	mongostorage.Load_MongoDB(&cfg)

	if flagRandomUpDN {
		go mongostorage.MongoWorker_UpDn_Random()
	}

	target := uint64(0)
	id := 0
	if flagTestPar >= 2 {
		// Do Testing heavily in parallel running go routines doing randomized stuff from 'testChaos'
		for i := 1; i <= flagTestPar; i++ {
			testCases = nil
			RandomStringSlice(testChaos)
			testCases = append(testCases, testChaos, testChaos, testChaos)
			RandomStringSlices(testCases)
			log.Printf("parallel run %d/%d testCases='%v'", i, flagTestPar, testCases)
			for _, testRun := range testCases {
				log.Printf("parallel run %d) testRun='%v'", i, testRun)
				//DEBUGSLEEP()
				for _, caseToTest := range testRun {
					if flagNumIterations > 0 {
						target += flagNumIterations
						id++
						go TestArticles(id, flagNumIterations, caseToTest, flagFormat, cfg.TestAfterInsert)
						// this launches a lot of go routines in the background for every appended testcase
						// possibly much more than allowed in -test-par, so many will block
						// and execution is randomized as possible
					}
				} // end for
			} // end for
		} // end for
	} else {
		// single threaded test with defined 'testCases'
		for _, testRun := range testCases {
			log.Printf("single testRun='%v'", testRun)
			for _, caseToTest := range testRun {
				DEBUGSLEEP()
				if flagNumIterations > 0 {
					target += flagNumIterations
					log.Printf("Start caseToTest=%s flagNumIterations=%d", caseToTest, flagNumIterations)
					id++
					TestArticles(id, flagNumIterations, caseToTest, flagFormat, cfg.TestAfterInsert)
					log.Printf("End caseToTest=%s flagNumIterations=%d", caseToTest, flagNumIterations)
				}
			} //end for
		} // end for
	} // end if flagTestPar

	log.Printf("wait to finish")

wait:

	for {
		time.Sleep(time.Second)
		if (len(pardonechan) == flagTestPar) || (flagTestPar <= 1 && len(pardonechan) == cap(pardonechan)) {
			r := mongostorage.Counter.Get("Did_mongoWorker_Reader")
			i := mongostorage.Counter.Get("Did_mongoWorker_Insert")
			d := mongostorage.Counter.Get("Did_mongoWorker_Delete")
			sum := r + i + d
			if sum == target {
				log.Printf("Test completed: %d/%d r=%d i=%d d=%d target=%d", len(pardonechan), cap(pardonechan), r, i, d, target)
				break wait
			}
			log.Printf("Waiting for Test to complete: %d/%d r=%d i=%d d=%d target=%d/%d", len(pardonechan), cap(pardonechan), r, i, d, sum, target)
			continue wait
		}
		log.Printf("Waiting for Test to complete: %d/%d flagTestPar=%d", len(pardonechan), cap(pardonechan), flagTestPar)
	} // end for

	log.Printf("Closing mongodbtest")
	time.Sleep(time.Second)
	close(mongostorage.Mongo_Delete_queue)
	close(mongostorage.Mongo_Insert_queue)
	close(mongostorage.Mongo_Reader_queue)
	time.Sleep(time.Second * 2)
	log.Printf("Quit mongodbtest")
} // end func main

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
func TestArticles(id int, NumIterations uint64, caseToTest string, flagFormat string, checkAfterInsert bool) {
	LockPar(&caseToTest)
	defer UnLockPar(&caseToTest)
	log.Printf("j=%d) TestArticles() run test: case '%s'", id, caseToTest)
	defer log.Printf("j=%d) TestArticles() end test: case '%s'", id, caseToTest)

	insreqs, delreqs, getreqs := 0, 0, 0
	t_get, t_ins, t_del, t_nf := 0, 0, 0, 0

	// get_retchan is used to receive the deleted count of articles back from worker when performing a delete operation.
	//del_retchan := make(chan int64, 1)

	// get_retchan is a buffered channel with a capacity of 1 to store slices of pointers to Mongostorage.MongoArticle objects.
	// get_retchan is used to send the retrieved articles back to the caller when performing a read operation.
	get_retchan := make(chan []*mongostorage.MongoArticle, 1)

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

		// test additional hashes for requests to test errors
		hash2 := "none2"
		hash3 := "none3"

		// Create the Article.
		article := mongostorage.MongoArticle{
			MessageIDHash: &messageIDHash,
			MessageID:     &messageID,
		}

		article.Head, article.Headsize = mongostorage.Strings2Byte(flagFormat, headerLines)
		article.Body, article.Bodysize = mongostorage.Strings2Byte(flagFormat, bodyLines)

		if article.Head == nil || article.Headsize <= 0 {
			log.Printf("IGNORE empty head=%v size=%d", article.Head, article.Headsize)
			continue
		}

		if article.Body == nil || article.Bodysize <= 0 {
			log.Printf("IGNORE empty body=%v size=%d", article.Body, article.Bodysize)
			continue
		}

		flagTestAdditional := false

		switch caseToTest {
		case "delete":
			//DEBUGSLEEP()

			t_del++
			delreqs++
			if delreqs >= 1000 {
				delreqs = 0
				log.Printf("j=%d) Add #%d/%d to Mongo_Delete_queue=%d/%d", id, t_del, NumIterations, len(mongostorage.Mongo_Delete_queue), cap(mongostorage.Mongo_Delete_queue))
			}

			log.Printf("j=%d) Add #%d to Mongo_Delete_queue=%d/%d", id, t_del, len(mongostorage.Mongo_Delete_queue), cap(mongostorage.Mongo_Delete_queue))
			// *?* passing a RetChan to delete request makes it somehow slower
			// *?* with RetChan and batchsize > 1: you have to wait for flushtimer or batchsize getting full
			delreq := mongostorage.MongoDelRequest{
				Msgidhashes: []*string{&messageIDHash},
				//RetChan:     del_retchan,
			}
			if flagTestAdditional {
				delreq.Msgidhashes = []*string{&messageIDHash, &hash2, &hash3}
			}

			// pass pointer to del request to Mongo_Reader_queue
			mongostorage.Mongo_Delete_queue <- &delreq

			// *?* deleted := <- del_retchan
			// *?* log.Printf("Test DELETE sent=%d, deleted=%d", len(delreq.Msgidhashes), deleted)

		case "read":
			//DEBUGSLEEP()
			getreqs++
			if getreqs >= 1000 {
				getreqs = 0
				log.Printf("j=%d) testCase: 'read' t_get=%d t_nf=%d itrerations=%d", id, t_get, t_nf, NumIterations)
			}
			log.Printf("j=%d) Add #%d to Mongo_Reader_queue=%d/%d", id, getreqs, len(mongostorage.Mongo_Reader_queue), cap(mongostorage.Mongo_Reader_queue))

			getreq := mongostorage.MongoGetRequest{
				Msgidhashes: []*string{&messageIDHash},
				RetChan:     get_retchan,
			}
			if flagTestAdditional {
				getreq.Msgidhashes = []*string{&messageIDHash, &hash2, &hash3}
			}
			// pass pointer to get request to Mongo_Reader_queue
			mongostorage.Mongo_Reader_queue <- &getreq

			log.Printf("j=%d) read waiting for reply on RetChan q=%d", id, len(mongostorage.Mongo_Reader_queue))
			select {
			case articles, ok := <-get_retchan:
				if !ok {
					log.Printf("INFO getreq.retchan has been closed")
				}
				if len(articles) == 0 {
					log.Printf("j=%d) testCase: 'read' empty reply from retchan hash='%s'", id, *article.MessageIDHash)
				} else {
					got_read := 0
					not_found := 0
					for _, article := range articles {
						if article.Found {
							switch article.Enc {
							case mongostorage.NOCOMP:
								// pass
							case mongostorage.GZIP_enc:
								if err := mongostorage.DecompressData(article.Head, mongostorage.GZIP_enc); err != nil {
									log.Printf("j=%d) GzipUncompress Head returned err: %v", id, err)
									continue
								}

								if err := mongostorage.DecompressData(article.Body, mongostorage.GZIP_enc); err != nil {
									log.Printf("j=%d) GzipUncompress Body returned err: %v", id, err)
									continue
								}

							case mongostorage.ZLIB_enc:
								if err := mongostorage.DecompressData(article.Head, mongostorage.ZLIB_enc); err != nil {
									log.Printf("j=%d) ZlibUncompress Head returned err: %v", id, err)
									continue
								}

								if err := mongostorage.DecompressData(article.Body, mongostorage.ZLIB_enc); err != nil {
									log.Printf("j=%d) ZlibUncompress Body returned err: %v", id, err)
									continue
								}

							} // end switch Enc decoder
							got_read++
							t_get++
							log.Printf("j=%d) testCase: 'read' t_get=%d got_read=%d/%d head='%s' body='%s' msgid='%s' hash=%s a.found=%t enc=%d", id, t_get, got_read, len(articles), string(*article.Head), string(*article.Body), *article.MessageID, *article.MessageIDHash, article.Found, article.Enc)
						} else {
							t_nf++
							not_found++
							log.Printf("j=%d) testCase: 'read' t_nf=%d articles_not_found=%d/%d hash=%s a.found=%t", id, t_nf, not_found, len(articles), *article.MessageIDHash, article.Found)
						}
					} // for article := range articles
				} // end if len(articles)
			} // end select

		case "no-compression":
			//DEBUGSLEEP()
			// Inserts the article into MongoDB without compression
			insreqs++
			t_ins++
			if insreqs >= 1000 {
				insreqs = 0
				log.Printf("j=%d) testCase: '%s' t_ins=%d/%d", id, caseToTest, t_ins, NumIterations)
			}
			log.Printf("j=%d) caseToTest=%s Add #%d to Mongo_Insert_queue=%d/%d", id, caseToTest, t_ins, len(mongostorage.Mongo_Insert_queue), cap(mongostorage.Mongo_Insert_queue))
			article.Enc = mongostorage.NOCOMP
			mongostorage.Mongo_Insert_queue <- &article

		case "gzip":
			//DEBUGSLEEP()
			// Inserts the article into MongoDB with gzip compression
			insreqs++
			t_ins++
			log.Printf("j=%d) caseToTest=%s BEFORE GZIP Headsize=%d Bodysize=%d hash=%s", id, caseToTest, article.Headsize, article.Bodysize, *article.MessageIDHash)
			if err := mongostorage.CompressData(article.Head, mongostorage.GZIP_enc); err != nil {
				log.Printf("j=%d) GzipCompress Head returned err: %v", id, err)
				continue
			}
			if err := mongostorage.CompressData(article.Body, mongostorage.GZIP_enc); err != nil {
				log.Printf("j=%d) GzipCompress Body returned err: %v", id, err)
				continue
			}
			log.Printf("j=%d) caseToTest=%s AFTER GZIP Headsize=%d Bodysize=%d hash=%s", id, caseToTest, article.Headsize, article.Bodysize, *article.MessageIDHash)
			article.Enc = mongostorage.GZIP_enc
			// pass pointer to insert gzip request to Mongo_Reader_queue
			mongostorage.Mongo_Insert_queue <- &article
			if insreqs >= 1000 {
				insreqs = 0
				log.Printf("testCase: '%s' t_ins=%d", caseToTest, t_ins)
			}

		case "zlib":
			//DEBUGSLEEP()
			// Inserts the article into MongoDB with zlib compression
			insreqs++
			t_ins++
			log.Printf("j=%d) caseToTest=%s BEFORE ZLIB Headsize=%d Bodysize=%d hash=%s", id, caseToTest, article.Headsize, article.Bodysize, *article.MessageIDHash)
			if err := mongostorage.CompressData(article.Head, mongostorage.ZLIB_enc); err != nil {
				log.Printf("j=%d) ZlibCompress Head returned err='%v'", id, err)
				continue
			}
			if err := mongostorage.CompressData(article.Body, mongostorage.ZLIB_enc); err != nil {
				log.Printf("j=%d) ZlibCompress Body returned err='%v'", id, err)
				continue
			}
			log.Printf("j=%d) caseToTest=%s AFTER ZLIB Headsize=%d Bodysize=%d hash=%s", id, caseToTest, article.Headsize, article.Bodysize, *article.MessageIDHash)
			article.Enc = mongostorage.ZLIB_enc
			// pass pointer to insert zlib request to Mongo_Reader_queue
			mongostorage.Mongo_Insert_queue <- &article
			if insreqs >= 1000 {
				insreqs = 0
				log.Printf("j=%d) testCase: '%s' t_ins=%d", id, caseToTest, t_ins)
			}

		default:
			log.Fatalf("j=%d) Invalid case to test: %s", id, caseToTest)
		} // end switch caseToTest
	}
} // end func TestArticles

func DEBUGSLEEP() { time.Sleep(time.Second * 1) }

// function written by AI.
func hashMessageID(messageID string) string {
	hash := sha256.New()
	hash.Write([]byte(messageID))
	return hex.EncodeToString(hash.Sum(nil))
} // end func hashMessageID

// function written by AI.
func RandomStringSlice(slice []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
} // end func RandomStringSlice

// function written by AI.
func RandomStringSlices(slices [][]string) {
	for _, slice := range slices {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(slice), func(i, j int) {
			slice[i], slice[j] = slice[j], slice[i]
		})
	}
} // end func RandomStringSlices

func logf(DEBUG bool, format string, a ...any) {
	if DEBUG {
		log.Printf(format, a...)
	}
} // end logf

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
