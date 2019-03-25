package util

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"log"
	"strconv"
	"strings"
)

var (
	awsElbClient *elbv2.ELBV2
	aliyunClient *slb.Client
	AWS_TYPE     = "aws"
	ALIYUN_TYPE  = "aliyun"
	yunConfig    = YunConfig{
		AwsRegion:          "",
		AwsAccessKey:       "",
		AwsAecretAccessKey: "",

		AliyunRegion:          "",
		AliyunAccessKey:       "",
		AliyunAecretAccessKey: "",
	}
)

type YunConfig struct {
	AwsRegion          string
	AwsAccessKey       string
	AwsAecretAccessKey string

	AliyunRegion          string
	AliyunAccessKey       string
	AliyunAecretAccessKey string
}

func OperatLb(operate string) (bool, string) {
	result := false
	operates := strings.Split(operate, " ")
	if len(operate) != 7 {
		return false, "args error"
	} else {
		yunType := operates[1]
		lbType := operates[2]
		lbOperate := operates[3]
		InstanceId := operates[4]
		LoadBalancerId := operates[5]
		port, _ := strconv.ParseInt(operates[6], 0, 64)

		yun := ""
		if yunType == "aws" {
			yun = AWS_TYPE
		} else if yunType == "aliyun" {
			yun = ALIYUN_TYPE
		}

		if lbOperate == "in" {
			result = AddBackendServer(LoadBalancerId, InstanceId, port, lbType, yun, yunConfig)
		} else if lbOperate == "out" {
			result = RemoveBackendServer(LoadBalancerId, InstanceId, port, lbType, yun, yunConfig)
		}
	}
	return result, "args error"
}

func buildClient(yun string, yunconfig YunConfig) {
	if strings.Contains(yun, AWS_TYPE) {
		sess := session.Must(session.NewSession())
		config := &aws.Config{
			Region: &yunconfig.AwsRegion,
			Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
				AccessKeyID:     yunconfig.AwsAccessKey,
				SecretAccessKey: yunconfig.AwsAecretAccessKey,
			}),
		}
		config.WithCredentialsChainVerboseErrors(true)
		awsElbClient = elbv2.New(sess, config)
	} else if strings.Contains(yun, ALIYUN_TYPE) {
		var err error
		// 创建slbClient实例
		aliyunClient, err = slb.NewClientWithAccessKey(
			yunconfig.AliyunRegion,          // 您的地域ID
			yunconfig.AliyunAccessKey,       // 您的AccessKey ID
			yunconfig.AliyunAecretAccessKey, // 您的AccessKey Secret
		)
		if err != nil {
			// 异常处理
			panic(err)
		}
	}
}

func AddBackendServer(LoadBalancerId string, InstanceId string, port int64, lbType string, yun string, yunconfig YunConfig) bool {
	if strings.Contains(yun, AWS_TYPE) {
		buildClient(yun, yunconfig)

		input := &elbv2.RegisterTargetsInput{
			//arn:aws-cn:elasticloadbalancing:cn-northwest-1:099573169643:targetgroup/sip-router-api-test/9a32752f9696cae4
			TargetGroupArn: aws.String(LoadBalancerId),
			Targets: []*elbv2.TargetDescription{
				{
					Id:   aws.String(InstanceId),
					Port: aws.Int64(port),
				},
			},
		}

		result, err := awsElbClient.RegisterTargets(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case elbv2.ErrCodeTargetGroupNotFoundException:
					fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
				case elbv2.ErrCodeTooManyTargetsException:
					fmt.Println(elbv2.ErrCodeTooManyTargetsException, aerr.Error())
				case elbv2.ErrCodeInvalidTargetException:
					fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
				case elbv2.ErrCodeTooManyRegistrationsForTargetIdException:
					fmt.Println(elbv2.ErrCodeTooManyRegistrationsForTargetIdException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			return false
		}

		fmt.Println(result)
		return true

	} else if strings.Contains(yun, ALIYUN_TYPE) {
		buildClient(yun, yunconfig)
		request := slb.CreateAddBackendServersRequest()
		request.BackendServers = fmt.Sprintf("[{\"ServerId\":\"%s\",\"Weight\":\"100\"}]", InstanceId)
		request.LoadBalancerId = LoadBalancerId
		request.Port = fmt.Sprint(port)
		response, err := aliyunClient.AddBackendServers(request)
		if err != nil {
			// 异常处理
			panic(err)
			return false
		}
		fmt.Printf("success(%d)! loadBalancerId = %s\n", response.GetHttpStatus(), response.LoadBalancerId)
		log.Println("BackendServers:", response.BackendServers)
		return true
	}
	return false
}

func RemoveBackendServer(LoadBalancerId string, InstanceId string, port int64, lbType string, yun string, yunconfig YunConfig) bool {
	count := DescribeTargetGroups(LoadBalancerId, lbType, yun, yunconfig)
	fmt.Println(count)
	if count <= 1 {
		log.Println("Instance count:", count, ",remove fail")
		return false
	}
	if strings.Contains(yun, AWS_TYPE) {
		buildClient(yun, yunconfig)
		input := &elbv2.DeregisterTargetsInput{
			TargetGroupArn: aws.String(LoadBalancerId),
			Targets: []*elbv2.TargetDescription{
				{
					Id:   aws.String(InstanceId),
					Port: aws.Int64(port),
				},
			},
		}

		result, err := awsElbClient.DeregisterTargets(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case elbv2.ErrCodeTargetGroupNotFoundException:
					fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
				case elbv2.ErrCodeInvalidTargetException:
					fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			return false
		}

		fmt.Println(result)
		return true
	} else if strings.Contains(yun, ALIYUN_TYPE) {
		buildClient(yun, yunconfig)
		request := slb.CreateRemoveBackendServersRequest()
		request.BackendServers = fmt.Sprintf("[\"%s\"]", InstanceId)
		request.LoadBalancerId = LoadBalancerId
		request.Port = fmt.Sprint(port)
		response, err := aliyunClient.RemoveBackendServers(request)
		if err != nil {
			// 异常处理
			fmt.Println(err, "Failed to remove backend servers, LoadBalancerId: "+InstanceId)
			panic(err)
			return false
		}
		fmt.Println(200, response.GetHttpStatus(), response.GetHttpContentString(), response.BackendServers)
		fmt.Println("success!")
		return true
	}
	return false
}

func DescribeTargetGroups(LoadBalancerId string, lbType string, yun string, yunconfig YunConfig) int {
	count := 0
	if strings.Contains(yun, AWS_TYPE) {
		buildClient(yun, yunconfig)
		input := &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: aws.String(LoadBalancerId),
		}

		result, err := awsElbClient.DescribeTargetHealth(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case elbv2.ErrCodeInvalidTargetException:
					fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
				case elbv2.ErrCodeTargetGroupNotFoundException:
					fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
				case elbv2.ErrCodeHealthUnavailableException:
					fmt.Println(elbv2.ErrCodeHealthUnavailableException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			return -1
		}

		for _, target := range result.TargetHealthDescriptions {
			if aws.StringValue(target.TargetHealth.State) == "healthy" {
				count++
			}
		}
		fmt.Println(result)

	} else if strings.Contains(yun, ALIYUN_TYPE) {
		buildClient(yun, yunconfig)
		request := slb.CreateDescribeHealthStatusRequest()
		request.LoadBalancerId = LoadBalancerId
		response, err := aliyunClient.DescribeHealthStatus(request)
		if err != nil {
			// 异常处理
			panic(err)
		}
		for _, backendServer := range response.BackendServers.BackendServer {
			if backendServer.ServerHealthStatus == "normal" {
				count++
			}
		}
		fmt.Println(200, response.GetHttpStatus(), response.GetHttpContentString(), response.BackendServers)
		fmt.Println("success!")
	}
	fmt.Println("healthy instance count:", count)
	return count
}
