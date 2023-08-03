# MongoDB Article Storage Test

This code is a test program for storing Usenet articles in MongoDB using the `mongostorage` package.

The main purpose of this test is to insert articles into the MongoDB collection based on different test cases.


## Prerequisites

Before running the test, make sure you have the following installed:

- Go programming language version 1.20 or above

## Installation

You can clone this repository and build the test program using the following commands:

```shell
git clone https://github.com/go-while/nntp-mongodb_test
cd nntp-mongodb_test
go build -o mongodbtest mongodbtest.go
```


## Usage

To run the test program, execute the compiled binary:

```shell
./mongodbtest # runs all defined 'testCases' in mongodbtest.go
```

```shell
# Run the 'delete' test case
./mongodbtest --test-case delete --test-num 1000

# Run the 'read' test case
./mongodbtest --test-case read --test-num 1000

# Run the 'no-compression' test case
./mongodbtest --test-case no-compression --test-num 1000

# Run the 'gzip' test case
./mongodbtest --test-case gzip --test-num 1000

# Run the 'zlib' test case
./mongodbtest --test-case zlib --test-num 1000
```


The test program will perform the following steps:
```
1. Load MongoDB storage with the default configurations.
2. Iterate over the specified number of inserts (default: 5000000).
3. For each insert, generate an example article with a unique MessageID and MessageIDHash.
4. Perform different actions based on the specified test case:
   - "no-compression": Insert articles without any compression.
   - "gzip": Insert articles with gzip compression.
   - "zlib": Insert articles with zlib compression.
   - "delete": Delete existing articles with the given MessageIDHash.
   - "read": Read existing articles with the given MessageIDHash.
5. Log details for each step, including the raw size of the article, the size after compression (if applicable), whether the article was inserted or read successfully, and the content of the article when the "read" test case is chosen.
```

## Test Cases
The test program supports the following test cases:

"no-compression": Articles are inserted without any compression.
"gzip": Articles are compressed using gzip before insertion.
"zlib": Articles are compressed using zlib before insertion.
"delete": Existing articles with the given MessageIDHash are deleted.
"read": Articles are read from the MongoDB collection based on the given MessageIDHash.

The test program will perform various operations based on the selected test case and log details for each step, including the raw size of the article, the size after compression (if applicable), whether the article was inserted or deleted successfully, and the content of the article when the "read" test case is chosen.

Please note that the "read" test case will retrieve the articles from the MongoDB collection, but it won't perform any verification on the content itself.

The purpose of the "read" test case is to demonstrate the retrieval of articles from the database. If you want to add further verification or validation of the content, you can modify the test program accordingly.


# ChatGPT Review Logical Considerations

Upon reviewing the code, I have identified a few potential logical issues or considerations:

The code has the following logical considerations and potential issues:

Command-Line Flag Validation: The code checks if the specified test case is a supported test case but lacks handling for invalid or unsupported test cases.

Iteration Count: The code runs multiple iterations of the test cases, but it does not handle cases where the number of iterations is negative or zero.

Error Handling: Detailed error handling is lacking for various operations, such as compression/decompression errors or MongoDB-related errors.

Time Delays: Time delays introduced using time.Sleep may not be optimally set for background operations.

Queue Size: The size of MongoDB queues is not explicitly set, which may impact performance depending on the expected load.

Hash Collisions: The code generates SHA256 hashes of message IDs, but it does not explicitly handle potential hash collisions.

TestCases Slice: The purpose of various test sequences in the testCases slice is not clear.

TestAfterInsert: The variable testAfterInsert is set to false, but its significance is unclear without further context.

Unused Variables: There are unused variables in the code.

Use of Select with RetChan: The code uses a select statement with RetChan but does not handle the case when the channel is closed, which may lead to a deadlock.

Message ID Handling: The code uses generated message IDs for testing, which may not follow real-world conventions.

Please note that these are observations and potential points for consideration. The code may have other context-specific requirements and logic that are not evident from the provided snippet. Further analysis and understanding of the entire system and its requirements would be necessary for a more comprehensive evaluation.

## Disclaimer

Please note that this test program and the module [https://github.com/go-while/nntp-mongodb-storage](https://github.com/go-while/nntp-mongodb-storage) were mostly generated by an AI language model (OpenAI's GPT-3.5, based on GPT-3) for demonstration purposes.

While efforts have been made to ensure its correctness, it is always recommended to review and validate the code before using it in production environments.

# License

This project is licensed under the MIT License - see the [LICENSE](https://choosealicense.com/licenses/mit/) file for details.


## Author
[go-while](https://github.com/go-while)
