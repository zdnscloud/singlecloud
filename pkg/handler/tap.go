package handler

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/golang/protobuf/ptypes/duration"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	sm "github.com/zdnscloud/servicemesh"
	pb "github.com/zdnscloud/servicemesh/public"
	"github.com/zdnscloud/singlecloud/hack/sockjs"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	TapApiURLPath = "/apis/tap.linkerd.io/v1alpha1/watch/namespaces/%s/%ss/%s/tap"
	DefaultMaxRps = 100.0
)

var (
	ValidTapResourceTypes = append(AppSupportWorkloadTypes, types.ResourceTypePod)
	ValidTapMethods       = []string{"POST", "GET", "PUT", "DELETE"}
)

func (m *ClusterManager) Tap(clusterID, ns, kind, name, toKind, toName, method, path string, r *http.Request, w http.ResponseWriter) {
	cluster := m.GetClusterByName(clusterID)
	if cluster == nil {
		log.Warnf("cluster %s isn't found to open tap", clusterID)
		return
	}

	req, err := buildTapRequest(ns, kind, name, toKind, toName, method, path)
	if err != nil {
		log.Warnf("build tap request failed: %s", err.Error())
		return
	}

	url, err := url.Parse(cluster.K8sConfig.Host)
	if err != nil {
		log.Warnf("build tap request url failed: %s", err.Error())
		return
	}

	url.Path = fmt.Sprintf(TapApiURLPath, ns, kind, name)
	fmt.Printf("url: %s\n", url.String())
	Sockjshandler := func(session sockjs.Session) {
		resp, err := sm.HandleRequest(cluster.KubeHttpClient, url, req)
		if err != nil {
			fmt.Printf("request get err: %v\n", err.Error())
			session.Close(503, err.Error())
			return
		}

		done := make(chan struct{})
		go func() {
			<-session.ClosedNotify()
			resp.Body.Close()
			close(done)
		}()

		reader := bufio.NewReader(resp.Body)
		for {
			event := pb.TapEvent{}
			err := sm.FromByteStreamToProtocolBuffers(reader, &event)
			if err == io.EOF {
				break
			}

			if err != nil {
				log.Warnf("receive tap response failed:%s", err.Error())
				break
			}

			tap, err := readerTapEvent(&event)
			if err != nil {
				log.Warnf("reader tap response failed:%s", err.Error())
				break
			}

			if err := session.Send(tap); err != nil {
				log.Warnf("send tap failed:%s", err.Error())
				break
			}
		}
		session.Close(503, "tap is terminated")
		<-done
	}

	tapPath := fmt.Sprintf(WSTapPathTemp, clusterID, ns, kind, name)
	sockjs.NewHandler(tapPath, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}

func buildTapRequest(namespace, kind, name, toKind, toName, method, path string) (*pb.TapByResourceRequest, error) {
	if slice.SliceIndex(ValidTapResourceTypes, kind) == -1 {
		return nil, fmt.Errorf("tap unsupported resource_type %s", kind)
	}

	matches := []*pb.TapByResourceRequest_Match{}
	if toKind != "" {
		if slice.SliceIndex(ValidTapResourceTypes, toKind) == -1 {
			return nil, fmt.Errorf("tap unsupported to_resource_type %s", toKind)
		}

		matches = append(matches, &pb.TapByResourceRequest_Match{
			Match: &pb.TapByResourceRequest_Match_Destinations{
				Destinations: &pb.ResourceSelection{
					Resource: &pb.Resource{
						Namespace: namespace,
						Type:      toKind,
						Name:      toName,
					},
				},
			},
		})
	}

	if method != "" {
		if slice.SliceIndex(ValidTapMethods, method) != -1 {
			return nil, fmt.Errorf("tap unsupported method: %s", method)
		}

		matches = append(matches, &pb.TapByResourceRequest_Match{
			Match: &pb.TapByResourceRequest_Match_Http_{
				Http: &pb.TapByResourceRequest_Match_Http{
					Match: &pb.TapByResourceRequest_Match_Http_Method{Method: method},
				},
			},
		})
	}

	if path != "" {
		matches = append(matches, &pb.TapByResourceRequest_Match{
			Match: &pb.TapByResourceRequest_Match_Http_{
				Http: &pb.TapByResourceRequest_Match_Http{
					Match: &pb.TapByResourceRequest_Match_Http_Path{Path: path},
				},
			},
		})
	}

	return &pb.TapByResourceRequest{
		Target: &pb.ResourceSelection{
			Resource: &pb.Resource{
				Namespace: namespace,
				Type:      kind,
				Name:      name,
			},
		},
		MaxRps: DefaultMaxRps,
		Match: &pb.TapByResourceRequest_Match{
			Match: &pb.TapByResourceRequest_Match_All{
				All: &pb.TapByResourceRequest_Match_Seq{
					Matches: matches,
				},
			},
		},
	}, nil
}

func readerTapEvent(pbEvent *pb.TapEvent) (string, error) {
	tap := types.Tap{
		Source:          getTcpAddr(pbEvent.GetSource()),
		SourceMeta:      types.EndpointMeta{Labels: pbEvent.GetSourceMeta().GetLabels()},
		Destination:     getTcpAddr(pbEvent.GetDestination()),
		DestinationMeta: types.EndpointMeta{Labels: pbEvent.GetDestinationMeta().GetLabels()},
		RouteMeta:       types.EndpointMeta{Labels: pbEvent.GetRouteMeta().GetLabels()},
		ProxyDirection:  pbEvent.GetProxyDirection().String(),
		Event: types.Event{
			RequestInit:  pbReqInitToScReqInit(pbEvent),
			ResponseInit: pbRespInitToScRespInit(pbEvent),
			ResponseEnd:  pbRespEndToScRespEnd(pbEvent),
		},
	}

	data, err := json.Marshal(&tap)
	return string(data), err
}

func getTcpAddr(pbTcpAddr *pb.TcpAddress) types.TcpAddress {
	return types.TcpAddress{
		Ip:   pbIPAddrToString(pbTcpAddr.GetIp()),
		Port: int(pbTcpAddr.GetPort()),
	}
}

func pbIPAddrToString(pbIp *pb.IPAddress) string {
	var b []byte
	if pbIp.GetIpv6() != nil {
		b = make([]byte, 16)
		binary.BigEndian.PutUint64(b[:8], pbIp.GetIpv6().GetFirst())
		binary.BigEndian.PutUint64(b[8:], pbIp.GetIpv6().GetLast())
	} else if pbIp.GetIpv4() != 0 {
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, pbIp.GetIpv4())
	}
	return net.IP(b).String()
}

func pbReqInitToScReqInit(pbEvent *pb.TapEvent) types.HttpRequestInit {
	pbReqInit := pbEvent.GetHttp().GetRequestInit()
	if pbReqInit == nil {
		return types.HttpRequestInit{}
	}

	return types.HttpRequestInit{
		Id:        pbHttpStreamIdToScHttpStreamId(pbReqInit.GetId()),
		Method:    pbMethodToString(pbReqInit.GetMethod()),
		Scheme:    pbSchemeToStirng(pbReqInit.GetScheme()),
		Authority: pbReqInit.GetAuthority(),
		Path:      pbReqInit.GetPath(),
		Headers:   pbHeadersToScHeaders(pbReqInit.GetHeaders()),
	}
}

func pbHttpStreamIdToScHttpStreamId(pbId *pb.TapEvent_Http_StreamId) types.HttpStreamId {
	return types.HttpStreamId{
		Base:   int(pbId.GetBase()),
		Stream: int(pbId.GetStream()),
	}
}

func pbMethodToString(pbMethod *pb.HttpMethod) string {
	if x, ok := pbMethod.GetType().(*pb.HttpMethod_Registered_); ok {
		return x.Registered.String()
	}

	if s, ok := pbMethod.GetType().(*pb.HttpMethod_Unregistered); ok {
		return s.Unregistered
	}

	return ""
}

func pbSchemeToStirng(pbScheme *pb.Scheme) string {
	if x, ok := pbScheme.GetType().(*pb.Scheme_Registered_); ok {
		return x.Registered.String()
	}

	if s, ok := pbScheme.GetType().(*pb.Scheme_Unregistered); ok {
		return s.Unregistered
	}

	return ""
}

func pbHeadersToScHeaders(pbHeaders *pb.Headers) []types.Header {
	var headers []types.Header
	for _, pbHeader := range pbHeaders.GetHeaders() {
		if s, ok := pbHeader.GetValue().(*pb.Headers_Header_ValueStr); ok {
			headers = append(headers, types.Header{
				Name:  pbHeader.GetName(),
				Value: s.ValueStr,
			})
		}
	}

	return headers
}

func pbRespInitToScRespInit(pbEvent *pb.TapEvent) types.HttpResponseInit {
	pbRespInit := pbEvent.GetHttp().GetResponseInit()
	if pbRespInit == nil {
		return types.HttpResponseInit{}
	}

	return types.HttpResponseInit{
		Id:               pbHttpStreamIdToScHttpStreamId(pbRespInit.GetId()),
		SinceRequestInit: pbDurationToScDuration(pbRespInit.GetSinceRequestInit()),
		HttpStatus:       int(pbRespInit.GetHttpStatus()),
		Headers:          pbHeadersToScHeaders(pbRespInit.GetHeaders()),
	}
}

func pbDurationToScDuration(pbDuration *duration.Duration) types.Duration {
	return types.Duration{
		Seconds: int(pbDuration.GetSeconds()),
		Nanos:   int(pbDuration.GetNanos()),
	}
}

func pbRespEndToScRespEnd(pbEvent *pb.TapEvent) types.HttpResponseEnd {
	pbRespEnd := pbEvent.GetHttp().GetResponseEnd()
	if pbRespEnd == nil {
		return types.HttpResponseEnd{}
	}

	return types.HttpResponseEnd{
		Id:                pbHttpStreamIdToScHttpStreamId(pbRespEnd.GetId()),
		SinceRequestInit:  pbDurationToScDuration(pbRespEnd.GetSinceRequestInit()),
		SinceResponseInit: pbDurationToScDuration(pbRespEnd.GetSinceResponseInit()),
		ResponseBytes:     int(pbRespEnd.GetResponseBytes()),
		Eos:               pbEosToInt(pbRespEnd.GetEos()),
		Trailers:          pbHeadersToScHeaders(pbRespEnd.GetTrailers()),
	}
}

func pbEosToInt(pbEos *pb.Eos) int {
	if i, ok := pbEos.GetEnd().(*pb.Eos_GrpcStatusCode); ok {
		return int(i.GrpcStatusCode)
	}

	if i, ok := pbEos.GetEnd().(*pb.Eos_ResetErrorCode); ok {
		return int(i.ResetErrorCode)
	}

	return 0
}
