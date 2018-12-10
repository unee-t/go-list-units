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
