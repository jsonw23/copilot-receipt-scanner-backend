# AWS Copilot Sample Application - Receipt Scanner
This is the back-end worker service for my receipt scanner copilot sample application.  This needs some configuration from [copilot-receipt-scanner](https://github.com/jsonw23/copilot-receipt-scanner) before it can be deployed, since Copilot doesn't support cross-stack dependency injection.

## Installation
To install this on your AWS account, you'll need:
- AWS Credentials configured for programmatic access.  Install the AWS CLI and use the [configure command](https://docs.aws.amazon.com/cli/latest/reference/configure/)
- [AWS Copilot CLI](https://aws.amazon.com/containers/copilot/)
- Docker

### Pre-Configuration
Get the load balancer endpoint URL from the front-end and add it to `copilot/image-handler/addons/image-status.yml` so that the SNS subscription can pass image status updates back to the API.

```
Mappings:
  EndpointMap:
    test:
      Url: http://{ELB_NAME}.{REGION}.elb.amazonaws.com/imageStatus
```

Next, get the name of the S3 bucket that was created by the front-end and add it to `copilot/image-handler/addons/textract-access.yml` so that the IAM access policy for Textract to detect text in the uploaded images can reference it.

```
Mappings:
  BucketMap:
    test:
      Bucket: "{BUCKET_NAME}"
```

These are in CloudFormation Mappings so that the application can have multiple environments set up without getting the buckets or API endpoints mixed up.  'test' is the name of the default environment you deploy first with Copilot.  If this were a real application, you would eventually push a 'prod' environment and you would not want production to access test resources by oversight.

### Build & Deploy
The copilot manifest files are already in place, but you'll need to run `copilot init` to start provisioning resources and config in your AWS account.  Copilot will not read the manifest file before asking for service name and task type in the guided process, so pass some extra arguments for the service name and task type to match with the manifest.  Name the app whatever you want, but use my service names.

```
copilot init -d ./Dockerfile -n image-handler -t "Worker Service" --deploy
```

### Confirm the SNS Subscription
After deployment is complete, you'll need to confirm the SNS subscription to the API endpoint.  Go to the AWS SNS Console and look for the new subscription that's listed as 'Pending Confirmation'.

Click 'Request Confirmation', and then get the logs for the front-end service with the easy copilot command:

```
copilot svc logs -n api
```

There should be a long url logged with `Action=ConfirmSubscription`.  Copy it, and back in the AWS Console, click 'Confirm subscription' on the pending subscription, and paste in that URL.
