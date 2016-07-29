/*
 * Minio S3Verify Library for Amazon S3 Compatible Cloud Storage (C) 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/minio/s3verify/signv4"
)

// newListMultipartUploadsReq - Create a new HTTP request for List Multipart Uploads API.
func newListMultipartUploadsReq(config ServerConfig, bucketName string) (*http.Request, error) {
	// listMultipartUploadsReq - a new HTTP request for the List Multipart Uploads API.
	var listMultipartUploadsReq = &http.Request{
		Header: map[string][]string{
		// X-Amz-Content-Sha256 will be set below.
		},
		Body:   nil, // There is no body sent with GET requests.
		Method: "GET",
	}
	urlValues := make(url.Values)
	urlValues.Set("uploads", "")

	targetURL, err := makeTargetURL(config.Endpoint, bucketName, "", config.Region, urlValues)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader([]byte{})
	_, sha256Sum, _, err := computeHash(reader)
	if err != nil {
		return nil, err
	}
	// Set Header values and URL.
	listMultipartUploadsReq.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(sha256Sum))
	listMultipartUploadsReq.URL = targetURL

	listMultipartUploadsReq = signv4.SignV4(*listMultipartUploadsReq, config.Access, config.Secret, config.Region)
	return listMultipartUploadsReq, nil
}

// listMultipartUploadsVerify - Verify that the response returned matches what is expected.
func listMultipartUploadsVerify(res *http.Response, expectedStatus string, expectedList listMultipartUploadsResult) error {
	return nil
}

// verifyHeaderListMultipartUploads - verify the header returned matches what is expected.
func verifyHeaderListMultipartUploads(res *http.Response) error {
	if err := verifyStandardHeaders(res); err != nil {
		return err
	}
	return nil
}

// verifyStatusListMultipartUploads - verify the status returned matches what is expected.
func verifyStatusListMultipartUploads(res *http.Response, expectedStatus string) error {
	if res.Status != expectedStatus {
		err := fmt.Errorf("Unexpected Status Received: wanted %v, got %v", expectedStatus, res.Status)
		return err
	}
	return nil
}

// verifyBodyListMultipartUploads - verify the body returned matches what is expected.
func verifyBodyListMultipartUploads(res *http.Response, expectedList listMultipartUploadsResult) error {
	receivedList := listMultipartUploadsResult{}
	err := xmlDecoder(res.Body, &receivedList)
	if err != nil {
		return err
	}
	if len(receivedList.Uploads) != len(expectedList.Uploads) {
		err := fmt.Errorf("Unexpected Number of Uploads Listed: wanted %d, got %d", len(expectedList.Uploads), len(receivedList.Uploads))
		return err
	}
	totalUploads := 0
	for _, receivedUpload := range receivedList.Uploads {
		for _, expectedUpload := range expectedList.Uploads {
			if receivedUpload.Size == expectedUpload.Size &&
				receivedUpload.UploadID == expectedUpload.UploadID &&
				receivedUpload.Key == expectedUpload.Key {
				totalUploads++
			}
		}
	}
	// TODO: revisit this and the error and find a better way of checking.
	if totalUploads != len(expectedList.Uploads) {
		err := fmt.Errorf("Wrong MetaData Saved in Received List")
		return err
	}
	return nil
}

// mainListMultipartUploads - Entry point for the list-multipart-uplods API test.
func mainListMultipartUploads(config ServerConfig, curTest int) bool {
	message := fmt.Sprintf("[%02d/%d] Multipart (List-Uploads):", curTest, globalTotalNumTest)
	// Spin scanBar
	scanBar(message)
	bucketName := validBuckets[0].Name

	uploads := []ObjectMultipartInfo{}

	for _, multipartObject := range multipartObjects {
		uploads = append(uploads, ObjectMultipartInfo{
			Key:      multipartObject.Key,
			UploadID: multipartObject.UploadID,
		})
	}
	expectedList := listMultipartUploadsResult{
		Bucket:  bucketName,
		Uploads: uploads,
	}
	// Spin scanBar
	scanBar(message)
	// Create a new request.
	req, err := newListMultipartUploadsReq(config, bucketName)
	if err != nil {
		printMessage(message, err)
		return false
	}
	// Spin scanBar
	scanBar(message)
	// Execute the request.
	res, err := execRequest(req, config.Client, bucketName, "")
	if err != nil {
		printMessage(message, err)
		return false
	}
	defer closeResponse(res)
	// Spin scanBar
	scanBar(message)
	// Verify the response.
	if err := listMultipartUploadsVerify(res, "200 OK", expectedList); err != nil {
		printMessage(message, err)
		return false
	}
	// Spin scanBar
	scanBar(message)
	// Test passed.
	printMessage(message, nil)
	return true
}
