---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ably_ingress_rule_mongodb Resource - terraform-provider-ably"
subcategory: ""
description: |-
  The ably_ingress_rule_mongodb resource sets up a MongoDB Integration Rule to stream document changes from a database collection over Ably.
---

# ably_ingress_rule_mongodb (Resource)

The `ably_ingress_rule_mongodb` resource sets up a MongoDB Integration Rule to stream document changes from a database collection over Ably.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `app_id` (String) The Ably application ID.
- `target` (Attributes) object (rule_source) (see [below for nested schema](#nestedatt--target))

### Optional

- `status` (String) The status of the rule. Rules can be enabled or disabled.

### Read-Only

- `id` (String) The rule ID.

<a id="nestedatt--target"></a>
### Nested Schema for `target`

Required:

- `collection` (String) What the connector should watch within the database. The connector only supports watching collections.
- `database` (String) The MongoDB Database Name
- `full_document` (String) Controls whether the full document should be included in the published change events. Full Document is not available by default in all types of change event. Possible values are `updateLookup`, `whenAvailable`, `off`. The default is `off`.
- `full_document_before_change` (String) Controls whether the full document before the change should be included in the change event. Full Document before change is not available on all types of change event. Possible values are `whenAvailable` or `off`. The default is `off`.
- `pipeline` (String) A MongoDB pipeline to pass to the Change Stream API. This field allows you to control which types of change events are published, and which channel the change event should be published to. The pipeline must set the _ablyChannel field on the root of the change event. It must also be a valid JSON array of pipeline operations.
- `primary_site` (String) The primary site that the connector will run in. You should choose a site that is close to your database.
- `url` (String) The connection string of your MongoDB instance. (e.g. mongodb://user:pass@myhost.com)

