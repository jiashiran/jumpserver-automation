package util

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"jumpserver-automation/logs"
	"strconv"
	"strings"
)

var (
	awsElbClient *elbv2.ELBV2
	awsLbClient  *elb.ELB
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

//LB aws alb in InstanceId LoadBalancerId port
func OperatLb(operate string) (bool, string) {
	result := false
	operates := strings.Split(operate, " ")
	logs.Logger.Info(operates)
	if len(operates) != 7 {
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
		logs.Logger.Info(lbOperate, yunType, lbType, LoadBalancerId, InstanceId, port, yun)
		if lbOperate == "in" {
			result = addBackendServer(LoadBalancerId, InstanceId, port, lbType, yun, yunConfig)
		} else if lbOperate == "out" {
			result = removeBackendServer(LoadBalancerId, InstanceId, port, lbType, yun, yunConfig)
		}
	}
	return result, "args error"
}

func LbINFO(operate string) string {
	operates := strings.Split(operate, " ")
	logs.Logger.Info(operates)
	if len(operates) != 4 {
		return "args error"
	} else {
		yunType := operates[1]
		lbType := operates[2]
		LoadBalancerIdOrName := operates[3]

		yun := ""
		if yunType == "aws" {
			yun = AWS_TYPE
		} else if yunType == "aliyun" {
			yun = ALIYUN_TYPE
		}

		_, des := describeTargetGroups(LoadBalancerIdOrName, lbType, yun, yunConfig)
		return des
	}

	return "args error"
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
		awsLbClient = elb.New(sess, config)
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

func addBackendServer(LoadBalancerIdOrName string, InstanceId string, port int64, lbType string, yun string, yunconfig YunConfig) bool {
	if strings.Contains(yun, AWS_TYPE) {
		buildClient(yun, yunconfig)
		if lbType == "alb" {
			input := &elbv2.RegisterTargetsInput{
				//arn:aws-cn:elasticloadbalancing:cn-northwest-1:099573169643:targetgroup/sip-router-api-test/9a32752f9696cae4
				TargetGroupArn: aws.String(LoadBalancerIdOrName), //LoadBalancerId
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
						logs.Logger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
					case elbv2.ErrCodeTooManyTargetsException:
						logs.Logger.Error(elbv2.ErrCodeTooManyTargetsException, aerr.Error())
					case elbv2.ErrCodeInvalidTargetException:
						logs.Logger.Error(elbv2.ErrCodeInvalidTargetException, aerr.Error())
					case elbv2.ErrCodeTooManyRegistrationsForTargetIdException:
						logs.Logger.Error(elbv2.ErrCodeTooManyRegistrationsForTargetIdException, aerr.Error())
					default:
						logs.Logger.Error(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					logs.Logger.Info(err.Error())
				}
				return false
			}

			logs.Logger.Info(result)
			return true
		} else if lbType == "elb" {
			input := &elb.RegisterInstancesWithLoadBalancerInput{
				Instances: []*elb.Instance{
					{
						InstanceId: aws.String(InstanceId),
					},
				},
				LoadBalancerName: aws.String(LoadBalancerIdOrName), //LoadBalancerName
			}

			result, err := awsLbClient.RegisterInstancesWithLoadBalancer(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case elb.ErrCodeAccessPointNotFoundException:
						logs.Logger.Error(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
					case elb.ErrCodeInvalidEndPointException:
						logs.Logger.Error(elb.ErrCodeInvalidEndPointException, aerr.Error())
					default:
						logs.Logger.Error(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					logs.Logger.Error(err.Error())
				}
				return false
			}

			logs.Logger.Info(result)
			return true
		}

	} else if strings.Contains(yun, ALIYUN_TYPE) {
		buildClient(yun, yunconfig)
		request := slb.CreateAddBackendServersRequest()
		request.BackendServers = fmt.Sprintf("[{\"ServerId\":\"%s\",\"Weight\":\"100\"}]", InstanceId)
		request.LoadBalancerId = LoadBalancerIdOrName
		//request.Port = fmt.Sprint(port)
		response, err := aliyunClient.AddBackendServers(request)
		if err != nil {
			// 异常处理
			panic(err)
			return false
		}
		logs.Logger.Infof("success(%d)! loadBalancerId = %s\n", response.GetHttpStatus(), response.LoadBalancerId)
		logs.Logger.Info("BackendServers:", response.BackendServers)
		return true
	}
	return false
}

func removeBackendServer(LoadBalancerIdOrName string, InstanceId string, port int64, lbType string, yun string, yunconfig YunConfig) bool {
	count, _ := describeTargetGroups(LoadBalancerIdOrName, lbType, yun, yunconfig)
	logs.Logger.Info(count)
	if count <= 1 {
		logs.Logger.Error("Instance count:", count, ",remove fail")
		return false
	}
	if strings.Contains(yun, AWS_TYPE) {
		buildClient(yun, yunconfig)
		if lbType == "alb" {
			input := &elbv2.DeregisterTargetsInput{
				TargetGroupArn: aws.String(LoadBalancerIdOrName),
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
						logs.Logger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
					case elbv2.ErrCodeInvalidTargetException:
						logs.Logger.Error(elbv2.ErrCodeInvalidTargetException, aerr.Error())
					default:
						logs.Logger.Error(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					logs.Logger.Error(err.Error())
				}
				return false
			}

			logs.Logger.Info(result)
			return true
		} else if lbType == "elb" {
			input := &elb.DeregisterInstancesFromLoadBalancerInput{
				Instances: []*elb.Instance{
					{
						InstanceId: aws.String(InstanceId),
					},
				},
				LoadBalancerName: aws.String(LoadBalancerIdOrName),
			}

			result, err := awsLbClient.DeregisterInstancesFromLoadBalancer(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case elb.ErrCodeAccessPointNotFoundException:
						logs.Logger.Error(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
					case elb.ErrCodeInvalidEndPointException:
						logs.Logger.Error(elb.ErrCodeInvalidEndPointException, aerr.Error())
					default:
						logs.Logger.Error(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					logs.Logger.Error(err.Error())
				}
				return false
			}

			logs.Logger.Info(result)
		}

	} else if strings.Contains(yun, ALIYUN_TYPE) {
		buildClient(yun, yunconfig)
		request := slb.CreateRemoveBackendServersRequest()
		request.BackendServers = fmt.Sprintf("[\"%s\"]", InstanceId)
		request.LoadBalancerId = LoadBalancerIdOrName
		//request.Port = fmt.Sprint(port)
		response, err := aliyunClient.RemoveBackendServers(request)
		if err != nil {
			// 异常处理
			logs.Logger.Error(err, "Failed to remove backend servers, LoadBalancerId: "+InstanceId)
			panic(err)
			return false
		}
		logs.Logger.Info(200, response.GetHttpStatus(), response.GetHttpContentString(), response.BackendServers)
		logs.Logger.Info("success!")
		return true
	}
	return false
}

func describeTargetGroups(LoadBalancerIdOrName string, lbType string, yun string, yunconfig YunConfig) (c int, str string) {
	count := 0
	if strings.Contains(yun, AWS_TYPE) {
		buildClient(yun, yunconfig)
		if lbType == "alb" {
			input := &elbv2.DescribeTargetHealthInput{
				TargetGroupArn: aws.String(LoadBalancerIdOrName),
			}

			result, err := awsElbClient.DescribeTargetHealth(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case elbv2.ErrCodeInvalidTargetException:
						logs.Logger.Error(elbv2.ErrCodeInvalidTargetException, aerr.Error())
					case elbv2.ErrCodeTargetGroupNotFoundException:
						logs.Logger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
					case elbv2.ErrCodeHealthUnavailableException:
						logs.Logger.Error(elbv2.ErrCodeHealthUnavailableException, aerr.Error())
					default:
						logs.Logger.Error(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					logs.Logger.Error(err.Error())
				}
				return -1, err.Error()
			}

			for _, target := range result.TargetHealthDescriptions {
				if aws.StringValue(target.TargetHealth.State) == elbv2.TargetHealthStateEnumHealthy {
					count++
				}
			}
			logs.Logger.Info(result)
			str = fmt.Sprint(result.TargetHealthDescriptions)
		} else if lbType == "elb" {
			input := &elb.DescribeInstanceHealthInput{
				LoadBalancerName: aws.String(LoadBalancerIdOrName),
			}

			result, err := awsLbClient.DescribeInstanceHealth(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case elb.ErrCodeAccessPointNotFoundException:
						logs.Logger.Error(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
					case elb.ErrCodeInvalidEndPointException:
						logs.Logger.Error(elb.ErrCodeInvalidEndPointException, aerr.Error())
					default:
						logs.Logger.Error(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					logs.Logger.Error(err.Error())
				}
				return -1, err.Error()
			}

			for _, target := range result.InstanceStates {
				if aws.StringValue(target.State) == "InService" {
					count++
				}
			}
			str = fmt.Sprint(result.InstanceStates)
			logs.Logger.Info(result)
		}

	} else if strings.Contains(yun, ALIYUN_TYPE) {
		buildClient(yun, yunconfig)
		request := slb.CreateDescribeHealthStatusRequest()
		request.LoadBalancerId = LoadBalancerIdOrName
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
		logs.Logger.Info(200, response.GetHttpStatus(), response.GetHttpContentString(), response.BackendServers)
		logs.Logger.Info("success!")
		str = fmt.Sprint(response.BackendServers)
	}
	logs.Logger.Info("healthy instance count:", count)
	return count, str
}
