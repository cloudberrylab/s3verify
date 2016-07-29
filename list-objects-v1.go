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
	"sort"

	"github.com/minio/s3verify/signv4"
)

// newListObjectsV1Req - Create a new HTTP request for ListObjects V1.
func newListObjectsV1Req(config ServerConfig, bucketName string, parameters map[string]string) (*http.Request, error) {
	// listObjectsV1Req - a new HTTP request for ListObjects V1.
	var listObjectsV1Req = &http.Request{
		Header: map[string][]string{
		// X-Amz-Content-Sha256 will be set below.
		},
		Body:   nil, // There is no body sent in GET requests.
		Method: "GET",
	}
	urlValues := make(url.Values)
	for k, v := range parameters {
		urlValues.Set(k, v)
	}
	targetURL, err := makeTargetURL(config.Endpoint, bucketName, "", config.Region, urlValues)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader([]byte{})
	_, sha256Sum, _, err := computeHash(reader)
	if err != nil {
		return nil, err
	}
	listObjectsV1Req.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(sha256Sum))
	listObjectsV1Req.URL = targetURL
	listObjectsV1Req = signv4.SignV4(*listObjectsV1Req, config.Access, config.Secret, config.Region)

	return listObjectsV1Req, nil
}

// listObjectsV1Verify - verify the response returned matches what is expected.
func listObjectsV1Verify(res *http.Response, expectedStatus string, expectedList listBucketResult) error {
	if err := verifyStatusListObjectsV1(res, expectedStatus); err != nil {
		return err
	}
	if err := verifyBodyListObjectsV1(res, expectedList); err != nil {
		return err
	}
	if err := verifyHeaderListObjectsV1(res); err != nil {
		return err
	}
	return nil
}

// verifyStatusListObjectsV1 - verify the status returned matches what is expected.
func verifyStatusListObjectsV1(res *http.Response, expectedStatus string) error {
	if res.Status != expectedStatus {
		err := fmt.Errorf("Unexpected Status Received: wanted %v, got %v", expectedStatus, res.Status)
		return err
	}
	return nil
}

// verifyBodyListObjectsV1 - verify the body returned matches what is expected.
func verifyBodyListObjectsV1(res *http.Response, expectedList listBucketResult) error {
	receivedList := listBucketResult{}
	err := xmlDecoder(res.Body, &receivedList)
	if err != nil {
		return err
	}
	if receivedList.Name != expectedList.Name {
		err := fmt.Errorf("Unexpected Bucket Listed: wanted %v, got %v", expectedList.Name, receivedList.Name)
		return err
	}
	listedObjects := 0
	for _, receivedObject := range receivedList.Contents {
		for _, expectedObject := range expectedList.Contents {
			if receivedObject.Key == expectedObject.Key &&
				receivedObject.Size == expectedObject.Size &&
				receivedObject.ETag == "\""+expectedObject.ETag+"\"" {
				listedObjects++
			}
		}
	}
	if expectedList.MaxKeys != 0 {
		if expectedList.MaxKeys != int64(len(receivedList.Contents)+len(receivedList.CommonPrefixes)) {
			err := fmt.Errorf("Unexpected Number of Objects Listed: wanted %d, got %d", expectedList.MaxKeys, len(receivedList.Contents)+len(receivedList.CommonPrefixes))
			return err
		}
	} else {
		if len(objects) != listedObjects {
			err := fmt.Errorf("Unexpected Number of Objects Listed: wanted %d, got %d", len(objects), listedObjects)
			return err
		}
	}
	return nil
}

// verifyHeaderListObjectsV1 - verify the header returned matches what is expected.
func verifyHeaderListObjectsV1(res *http.Response) error {
	if err := verifyStandardHeaders(res); err != nil {
		return err
	}
	return nil
}

// mainListObjectsV1 - Entry point for the ListObjects V1 API test.
func mainListObjectsV1(config ServerConfig, curTest int) bool {
	message := fmt.Sprintf("[%02d/%d] ListObjects V1:", curTest, globalTotalNumTest)
	// Spin scanBar
	scanBar(message)
	bucketName := validBuckets[0].Name
	objectInfo := ObjectInfos{}
	for _, object := range objects {
		objectInfo = append(objectInfo, *object)
	}
	// Order objects by their Key.
	sort.Sort(objectInfo)
	// Test for listobjects with no extra parameters.
	expectedList := listBucketResult{
		Name:     bucketName, // Listing from the first bucket created that houses all objects.
		Contents: objectInfo, // The first bucket created will house all the objects created by the PUT object test.
		// Currently the ListObjects V1 test does not test with extra 'parameters' set:
		// Prefix
		// Max-Keys
		// Marker
		// Delimiter
	}
	// Test for listobjects with maxkeys parameter set.
	expectedListMaxKeys := listBucketResult{
		Name:     bucketName,
		Contents: objectInfo[:31], // Only return the first 30 objects.
		MaxKeys:  30,              // Only return the first 30 objects.
	}
	// Store the parameters to be set by the request.
	maxKeysMap := map[string]string{
		"max-keys": "30", // 30 objects.
	}
	// Create a new request.
	noParamReq, err := newListObjectsV1Req(config, bucketName, nil) // No extra parameters for the first test.
	if err != nil {
		printMessage(message, err)
		return false
	}
	// Spin scanBar
	scanBar(message)
	// Execute the request.
	noParamRes, err := execRequest(noParamReq, config.Client, bucketName, "")
	if err != nil {
		printMessage(message, err)
		return false
	}
	defer closeResponse(noParamRes)
	// Spin scanBar
	scanBar(message)
	// Verify the response.
	if err := listObjectsV1Verify(noParamRes, "200 OK", expectedList); err != nil {
		printMessage(message, err)
		return false
	}
	// Spin scanBar
	scanBar(message)
	// Create a new request with max-keys set to 30.
	maxKeysReq, err := newListObjectsV1Req(config, bucketName, maxKeysMap) // MaxKeys set to 30.
	if err != nil {
		printMessage(message, err)
		return false
	}
	// Spin scanBar
	scanBar(message)
	// Execute the request.
	maxKeysRes, err := execRequest(maxKeysReq, config.Client, bucketName, "")
	if err != nil {
		printMessage(message, err)
		return false
	}
	defer closeResponse(maxKeysRes)
	// Spin scanBar
	scanBar(message)
	// Verify the max-keys parameter is respected.
	if err := listObjectsV1Verify(maxKeysRes, "200 OK", expectedListMaxKeys); err != nil {
		printMessage(message, err)
		return false
	}
	// Spin scanBar
	scanBar(message)
	// Test passed.
	printMessage(message, nil)
	return true
}
