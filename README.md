# IAM Auth policy

	{
	  "Version": "2012-10-17",
	  "Statement": [
		{
		  "Effect": "Allow",
		  "Action": [
			"rds-db:connect"
		  ],
		  "Resource": [
			"arn:aws:rds-db:ap-southeast-1:812644853088:dbuser:*/mydbuser"
		  ]
		}
	  ]
	}

https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/UsingWithRDS.IAMDBAuth.IAMPolicy.html
is misleading at time of writing. It should be correct to look like the above
example.
