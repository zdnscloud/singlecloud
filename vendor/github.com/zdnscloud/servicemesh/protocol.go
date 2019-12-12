package servicemesh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/golang/protobuf/proto"

	pb "github.com/zdnscloud/servicemesh/public"
)

const ErrorHeader = "linkerd-error"

func HandleApiRequest(serverUrl *url.URL, endpoint string, req proto.Message, resp proto.Message) error {
	httpResp, err := HandleRequest(http.DefaultClient, endpointNameToPublicAPIURL(serverUrl, endpoint), req)
	if err != nil {
		return err
	}

	defer httpResp.Body.Close()
	reader := bufio.NewReader(httpResp.Body)
	return FromByteStreamToProtocolBuffers(reader, resp)
}

func endpointNameToPublicAPIURL(serverUrl *url.URL, endpoint string) *url.URL {
	return serverUrl.ResolveReference(&url.URL{Path: endpoint})
}

func HandleRequest(client *http.Client, url *url.URL, req proto.Message) (*http.Response, error) {
	resp, err := post(client, url, req)
	if err != nil {
		return nil, fmt.Errorf("post request failed: %s", err.Error())
	}

	if err := checkIfResponseHasError(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func post(client *http.Client, url *url.URL, req proto.Message) (*http.Response, error) {
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %s", err.Error())
	}

	httpReq, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("new http request failed: %s", err.Error())
	}

	return client.Do(httpReq.WithContext(context.TODO()))
}

func checkIfResponseHasError(rsp *http.Response) error {
	errorMsg := rsp.Header.Get(ErrorHeader)
	if errorMsg != "" {
		reader := bufio.NewReader(rsp.Body)
		var apiError pb.ApiError
		err := FromByteStreamToProtocolBuffers(reader, &apiError)
		if err != nil {
			return fmt.Errorf("response has %s header [%s], but response body didn't contain protobuf error: %v",
				ErrorHeader, errorMsg, err)
		}

		return fmt.Errorf("response get error: %s", apiError.Error)
	}

	if rsp.StatusCode != http.StatusOK {
		if rsp.Body != nil {
			bytes, err := ioutil.ReadAll(rsp.Body)
			if err == nil && len(bytes) > 0 {
				return fmt.Errorf("http error, status code [%d] (unexpected api response: %s)", rsp.StatusCode, string(bytes))
			}
		}

		return fmt.Errorf("http error, status code [%d] (unexpected api response)", rsp.StatusCode)
	}

	return nil
}

func FromByteStreamToProtocolBuffers(byteStreamContainingMessage *bufio.Reader, out proto.Message) error {
	messageAsBytes, err := deserializePayloadFromReader(byteStreamContainingMessage)
	if err != nil {
		return fmt.Errorf("reading byte stream header failed: %s", err.Error())
	}

	err = proto.Unmarshal(messageAsBytes, out)
	if err != nil {
		return fmt.Errorf("unmarshalling array of [%d] bytes failed: %s", len(messageAsBytes), err.Error())
	}

	return nil
}

func deserializePayloadFromReader(reader *bufio.Reader) ([]byte, error) {
	messageLengthAsBytes := make([]byte, 4)
	_, err := io.ReadFull(reader, messageLengthAsBytes)
	if err != nil {
		return nil, fmt.Errorf("reading message length failed: %s", err.Error())
	}
	messageLength := int(binary.LittleEndian.Uint32(messageLengthAsBytes))

	messageContentsAsBytes := make([]byte, messageLength)
	_, err = io.ReadFull(reader, messageContentsAsBytes)
	if err != nil {
		return nil, fmt.Errorf("reading bytes from message failed: %s", err.Error())
	}

	return messageContentsAsBytes, nil
}
