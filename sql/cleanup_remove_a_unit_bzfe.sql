	SET @product_id = '%d';
	SET @script = 'cleanup_remove_a_unit_bzfe_v3.0.sql';

# Timestamp	
	SET @timestamp = NOW();

# Invitation information
	DELETE
	FROM `ut_invitation_api_data`
	WHERE (`bz_unit_id` = @product_id)
	;
	
# Information about the mapping user and unit
	DELETE
	FROM `ut_map_user_unit_details`
	WHERE (`bz_unit_id` = @product_id)
	;

# Flags
# Needs bugs information
	# Delete the flags associated to the bugs associated to that product in the 'flags' table
		DELETE `flags` 
		FROM `flags`
			INNER JOIN `bugs` 
				ON (`flags`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# We need to create a temp table until we cleanup the flag inclusion, exclusions and flagtypes
	# This is needed as this is where the link flagtyp/product_id are kept...
		CREATE TEMPORARY TABLE IF NOT EXISTS `flaginclusions_temp` AS (SELECT * FROM `flaginclusions`);
		CREATE TEMPORARY TABLE IF NOT EXISTS `flagexclusions_temp` AS (SELECT * FROM `flagexclusions`);

	# Delete the flag exclusion for flags related to this product in the 'flagexclusions' table
		DELETE  
		FROM `flagexclusions`
		WHERE (`flagexclusions`.`product_id` = @product_id);

	# Delete the flag inclusion for flags related to this product in the 'flaginclusions' table
		DELETE  
		FROM `flaginclusions`
		WHERE (`flaginclusions`.`product_id` = @product_id);
	
	# Delete the falgtypes associated to this product in the table 'flagtypes'
	# Step 1
	# We use the temp table for that
		DELETE `flagtypes`
		FROM
	    `flagtypes`
	    INNER JOIN `flaginclusions_temp` 
		ON (`flagtypes`.`id` = `flaginclusions_temp`.`type_id`)
		WHERE (`flaginclusions_temp`.`product_id` = @product_id);
	
	# Delete the falgtypes associated to this product in the table 'flagtypes'
	# Step 2 (to be thourough...)
	# We use the temp table for that
		DELETE `flagtypes`
		FROM
	    `flagtypes`
	    INNER JOIN `flagexclusions_temp` 
		ON (`flagtypes`.`id` = `flagexclusions_temp`.`type_id`)
		WHERE (`flagexclusions_temp`.`product_id` = @product_id);
		
	# Cleanup: we do not need the temp tables anymore:
		DROP TABLE IF EXISTS `flaginclusions_temp`;
		DROP TABLE IF EXISTS `flagexclusions_temp`;

# Tags
# Needs bugs and longdescs information
		
	# The tags for longdesc
		DELETE `longdescs_tags`
		FROM
		`longdescs_tags`
			INNER JOIN `longdescs` 
				ON (`longdescs_tags`.`comment_id` = `longdescs`.`comment_id`)
			INNER JOIN `bugs` 
				ON (`longdescs`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# The activity for the tags for longdesc
		DELETE `longdescs_tags_activity`
		FROM
		`longdescs_tags_activity`
			INNER JOIN `longdescs` 
				ON (`longdescs_tags_activity`.`comment_id` = `longdescs`.`comment_id`)
			INNER JOIN `bugs` 
				ON (`longdescs`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# Delete all the records for the tags associated to the bugs associated to this unit in the 'bug_tag' table
		DELETE `bug_tag` 
		FROM `bug_tag`
			INNER JOIN `bugs` 
				ON (`bug_tag`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

# Keywords
# Needs bug info
	# The link between keyworddefs and bugs for bugs associated to this product
		DELETE `keywords`
		FROM
		`keywords`
			INNER JOIN `bugs` 
				ON (`keywords`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

# The tables we use to process invitations and stuff
# Needs bug info
	
	# Add a user to a case 'ut_data_to_add_user_to_a_case'
		DELETE `ut_data_to_add_user_to_a_case`
		FROM
			`ut_data_to_add_user_to_a_case`
			INNER JOIN `bugs` 
				ON (`ut_data_to_add_user_to_a_case`.`bz_case_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# Add a user to a role in the unit 'ut_data_to_add_user_to_a_role'
		DELETE FROM `ut_data_to_add_user_to_a_role`
		WHERE `bz_unit_id` = @product_id;
	
	# Replace a dummy user with a 'real' user in a role in the unit 'ut_data_to_replace_dummy_roles'
		DELETE FROM `ut_data_to_replace_dummy_roles`
		WHERE `bz_unit_id` = @product_id;
		
# Bug/case related info 

	# Delete the Attach data if they exist:
		DELETE `attach_data` 
		FROM `attach_data`
			INNER JOIN `attachments` 
				ON (`attach_data`.`id` = `attachments`.`attach_id`)
			INNER JOIN `bugs` 
				ON (`attachments`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

	# Delete the Attachments if they exist:
		DELETE `attachments` 
		FROM `attachments`
			INNER JOIN `bugs` 
				ON (`attachments`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

	# Delete the roadbook for info if they exist:
		DELETE `bug_cf_ipi_clust_3_roadbook_for` 
		FROM `bug_cf_ipi_clust_3_roadbook_for`
			INNER JOIN `bugs` 
				ON (`bug_cf_ipi_clust_3_roadbook_for`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# Delete the accounting action if they exist:
		DELETE `bug_cf_ipi_clust_9_acct_action` 
		FROM `bug_cf_ipi_clust_9_acct_action`
			INNER JOIN `bugs` 
				ON (`bug_cf_ipi_clust_9_acct_action`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
	
	# Delete all the records for the bugs associated to this unit in the 'bug_group_map' table
		DELETE `bug_group_map` 
		FROM `bug_group_map`
			INNER JOIN `bugs` 
				ON (`bug_group_map`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

	# Delete all the records for the bugs associated to this unit in the 'bug_group_see_also' table
	#########
	#
	# WIP - WARNING - The below query only does 1/2 the work, we also need to remove the records where a 
	#	bug for this product/unit is referenced in the `value` field for this table
	# 
	#########
		DELETE `bug_see_also` 
		FROM `bug_see_also`
			INNER JOIN `bugs` 
				ON (`bug_see_also`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

	# Delete all the records for the last visit of a user to the bugs associated to this unit in the 'bug_user_last_visit' table
		DELETE `bug_user_last_visit` 
		FROM `bug_user_last_visit`
			INNER JOIN `bugs` 
				ON (`bug_user_last_visit`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

	# Delete all the records for the bug activity for the bugs associated to this unit in the 'bugs_activity' table
		DELETE `bugs_activity` 
		FROM `bugs_activity`
			INNER JOIN `bugs` 
				ON (`bugs_activity`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

	# Delete all the records for the bug aliases for the bugs associated to this unit in the 'bugs_aliases' table
		DELETE `bugs_aliases` 
		FROM `bugs_aliases`
			INNER JOIN `bugs` 
				ON (`bugs_aliases`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

	# Delete all the records for the fulltext of the bugs associated to this unit in the 'bugs_fulltext' table
		DELETE `bugs_fulltext` 
		FROM `bugs_fulltext`
			INNER JOIN `bugs` 
				ON (`bugs_fulltext`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
	
	# Delete all the records of the users in CC for the bugs associated to this unit in the 'cc' table
		DELETE `cc` 
		FROM `cc`
			INNER JOIN `bugs` 
				ON (`cc`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# Delete all the dependendies for bugs associated to this unit in the 'dependencies' table
	#	Step 1: blocks
		DELETE `dependencies` 
		FROM `dependencies`
			INNER JOIN `bugs` 
				ON (`dependencies`.`blocked` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# Delete all the dependendies for bugs associated to this unit in the 'dependencies' table
	#	Step 2: Depends On
		DELETE `dependencies` 
		FROM `dependencies`
			INNER JOIN `bugs` 
				ON (`dependencies`.`dependson` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# Delete all the duplicates for bugs associated to this unit in the 'duplicates' table
	#	Step 1: Dupe Of
		DELETE `duplicates` 
		FROM `duplicates`
			INNER JOIN `bugs` 
				ON (`duplicates`.`dupe_of` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);
		
	# Delete all the duplicates for bugs associated to this unit in the 'duplicates' table
	#	Step 2: Dupe
		DELETE `duplicates` 
		FROM `duplicates`
			INNER JOIN `bugs` 
				ON (`duplicates`.`dupe` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);

	# Delete all the email bug ignore data for bugs associated to this unit in the 'email_bug_ignore' table
		DELETE `email_bug_ignore` 
		FROM `email_bug_ignore`
			INNER JOIN `bugs` 
				ON (`email_bug_ignore`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);	

	# Delete all the longdescs for bugs associated to this unit in the 'longdescs' table
		DELETE `longdescs` 
		FROM `longdescs`
			INNER JOIN `bugs` 
				ON (`longdescs`.`bug_id` = `bugs`.`bug_id`)
		WHERE (`bugs`.`product_id` = @product_id);	
	
	# Delete all the bugs/cases associated to that product/unit
	# We need to do this LAST when we have no need for a link bug/product
		DELETE FROM `bugs`
		WHERE `product_id` = @product_id;

# Groups
	# Delete the Group Control Map: table 'group_control_map'
		DELETE FROM `group_control_map`
		WHERE `product_id` = @product_id;
		
	# Delete the permissions for the groups associated to that product: `group_group_map` table
	# Step 1: member_id
		DELETE `group_group_map`
		FROM
		`group_group_map`
			INNER JOIN `ut_product_group` 
				ON (`group_group_map`.`member_id` = `ut_product_group`.`group_id`)
		WHERE (`ut_product_group`.`product_id` = @product_id);
	
	# Delete the permissions for the groups associated to that product: `group_group_map` table
	# Step 2: grantor_id
		DELETE `group_group_map`
		FROM
		`group_group_map`
			INNER JOIN `ut_product_group` 
				ON (`group_group_map`.`grantor_id` = `ut_product_group`.`group_id`)
		WHERE (`ut_product_group`.`product_id` = @product_id);
		
	# Delete the permissions for the users for that product in the table 'user_group_map'
		DELETE `user_group_map` 
		FROM
		`user_group_map`
			INNER JOIN `ut_product_group` 
				ON (`user_group_map`.`group_id` = `ut_product_group`.`group_id`)
		WHERE (`ut_product_group`.`product_id` = @product_id);
	
	# Delete the groups associated to this product in the table 'groups'
		DELETE `groups` 
		FROM
		`groups`
			INNER JOIN `ut_product_group` 
				ON (`groups`.`id` = `ut_product_group`.`group_id`)
		WHERE (`ut_product_group`.`product_id` = @product_id);
	
# Components

	# Delete all the records of the user in associated to a component for that unit in the 'component_cc' table
		DELETE `component_cc` 
		FROM
		`component_cc`
		INNER JOIN `components` 
			ON (`component_cc`.`component_id` = `components`.`id`)
		WHERE (`components`.`product_id` = @product_id);

	# Delete the components associated to this product
		DELETE FROM `components`
		WHERE `product_id` = @product_id;

# Products:
	
	# Delete the milestone
		DELETE FROM `milestones`
		WHERE `product_id` = @product_id;
	
	# Delete the version
		DELETE FROM `versions`
		WHERE `product_id` = @product_id;
	
	# Delete the product
		DELETE FROM `products`
		WHERE `id` = @product_id;
	
# Cleanup 

	#Delete the records in the table `ut_product_group`
		DELETE FROM `ut_product_group`
			WHERE `product_id` = @product_id;

# Log 

	# Update the table 'ut_data_to_create_units' so that we record that the unit has been deleted
		UPDATE `ut_data_to_create_units`
		SET 
			`deleted_datetime` = @timestamp
			, `deletion_script` = @script
		WHERE `product_id` = @product_id;
