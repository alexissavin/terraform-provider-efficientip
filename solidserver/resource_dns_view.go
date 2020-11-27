package solidserver

import (
	//"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	//	"github.com/hashicorp/terraform/helper/validation"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func resourcednsview() *schema.Resource {
	return &schema.Resource{
		Create: resourcednsviewCreate,
		Read:   resourcednsviewRead,
		Update: resourcednsviewUpdate,
		Delete: resourcednsviewDelete,
		Exists: resourcednsviewExists,
		Importer: &schema.ResourceImporter{
			State: resourcednsviewImportState,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the DNS view to create.",
				Required:    true,
				ForceNew:    true,
			},
			"dnsserver": {
				Type:        schema.TypeString,
				Description: "The name of DNS server or DNS SMART hosting the DNS view to create.",
				Required:    true,
				ForceNew:    true,
			},
			"order": {
				Type:        schema.TypeInt,
				Description: "The level of the DNS view, where 0 represents the highest level in the views hierarchy.",
				Computed:    true,
				ForceNew:    false,
			},
			"recursion": {
				Type:        schema.TypeBool,
				Description: "The recursion mode of the DNS view (Default: true).",
				Optional:    true,
				Default:     false,
			},
			/* To Be Implemented Later (Requires the use of dns_view_param_add)
			"forward": {
				Type:        schema.TypeString,
				Description: "The forwarding mode of the DNS SMART (Supported: none, first, only; Default: none).",
				Optional:    true,
				Default:     "",
			},
			"forwarders": {
				Type:        schema.TypeList,
				Description: "The IP address list of the forwarder(s) configured to configure on the DNS SMART.",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			*/

			// ACL(s)
			// Views and Servers/SMARTs
			// allow_query
			// allow_recursion
			// allow_transfer

			// Views Only
			"match_clients": {
				Type:        schema.TypeList,
				Description: "A list of ACL name(s) or netork prefixes used to match the clients of the view.",
				Optional:    true,
				ForceNew:    false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"match_to": {
				Type:        schema.TypeList,
				Description: "A list of ACL name(s) or netork prefixes used to match the traffic to the view.",
				Optional:    true,
				ForceNew:    false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"class": {
				Type:        schema.TypeString,
				Description: "The class associated to the DNS server.",
				Optional:    true,
				ForceNew:    false,
				Default:     "",
			},
			"class_parameters": {
				Type:        schema.TypeMap,
				Description: "The class parameters associated to the view.",
				Optional:    true,
				ForceNew:    false,
				Default:     map[string]string{},
			},
		},
	}
}

func resourcednsviewExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("dnsview_id", d.Id())

	log.Printf("[DEBUG] Checking existence of DNS view (oid): %s\n", d.Id())

	// Sending read request
	resp, body, err := s.Request("get", "rest/dns_view_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			return true, nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				log.Printf("[DEBUG] SOLIDServer - Unable to find DNS view (oid): %s (%s)\n", d.Id(), errMsg)
			}
		} else {
			log.Printf("[DEBUG] SOLIDServer - Unable to find DNS view (oid): %s\n", d.Id())
		}

		// Unset local ID
		d.SetId("")
	}

	// Reporting a failure
	return false, err
}

func resourcednsviewCreate(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("add_flag", "new_only")
	parameters.Add("dnsview_name", strings.ToLower(d.Get("name").(string)))
	parameters.Add("dns_name", strings.ToLower(d.Get("dnsserver").(string)))

	// Configure recursion
	if d.Get("recursion").(bool) {
		parameters.Add("dnsview_recursion", "yes")
	} else {
		parameters.Add("dnsview_recursion", "no")
	}

	parameters.Add("dnsview_class_name", d.Get("class").(string))
	parameters.Add("dnsview_class_parameters", urlfromclassparams(d.Get("class_parameters")).Encode())

	// Sending creation request
	resp, body, err := s.Request("post", "rest/dns_view_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				log.Printf("[DEBUG] SOLIDServer - Created DNS view (oid): %s\n", oid)
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return fmt.Errorf("SOLIDServer - Unable to create DNS view: %s (%s)", strings.ToLower(d.Get("name").(string)), errMsg)
			}
		}

		return fmt.Errorf("SOLIDServer - Unable to create DNS view: %s\n", strings.ToLower(d.Get("name").(string)))
	}

	// Reporting a failure
	return err
}

//<FIXME OFFSET>

func resourcednsviewUpdate(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("dnsview_id", d.Id())
	parameters.Add("add_flag", "edit_only")

	// Configure recursion
	if d.Get("recursion").(bool) {
		parameters.Add("dnsview_recursion", "yes")
	} else {
		parameters.Add("dnsview_recursion", "no")
	}

	parameters.Add("dnsview_class_name", d.Get("class").(string))
	parameters.Add("dnsview_class_parameters", urlfromclassparams(d.Get("class_parameters")).Encode())

	// Sending the update request
	resp, body, err := s.Request("put", "rest/dns_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				log.Printf("[DEBUG] SOLIDServer - Updated DNS view (oid): %s\n", oid)
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return fmt.Errorf("SOLIDServer - Unable to update DNS view: %s (%s)", strings.ToLower(d.Get("name").(string)), errMsg)
			}
		}

		return fmt.Errorf("SOLIDServer - Unable to update DNS view: %s\n", strings.ToLower(d.Get("name").(string)))
	}

	// Reporting a failure
	return err
}

func resourcednsviewDelete(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	for i := 0; i < 3; i++ {
		// Building parameters
		parameters := url.Values{}
		parameters.Add("dnsview_id", d.Id())

		// Sending the deletion request
		resp, body, err := s.Request("delete", "rest/dns_view_delete", &parameters)

		if err == nil {
			var buf [](map[string]interface{})
			json.Unmarshal([]byte(body), &buf)

			// Checking the answer
			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				// Log deletion
				log.Printf("[DEBUG] SOLIDServer - Deleted DNS view (oid): %s\n", d.Id())

				// Unset local ID
				d.SetId("")

				// Reporting a success
				return nil
			} else {
				// Logging a failure
				if len(buf) > 0 {
					if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
						log.Printf("SOLIDServer - Unable to delete DNS view: %s (%s)", strings.ToLower(d.Get("name").(string)), errMsg)
					}
				} else {
					log.Printf("SOLIDServer - Unable to delete DNS view: %s", strings.ToLower(d.Get("name").(string)))
				}
				time.Sleep(time.Duration(8 * time.Second))
			}
		} else {
			// Reporting a failure
			return err
		}
	}

	// Reporting a failure
	return fmt.Errorf("SOLIDServer - Unable to delete DNS view: Too many unsuccessful deletion attempts")
}

func resourcednsviewRead(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("dnsview_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/dns_view_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("name", strings.ToLower(buf[0]["dnsview_name"].(string)))
			d.Set("dnsserver", buf[0]["dns_name"].(string))

			viewOrder, _ := strconv.Atoi(buf[0]["dnsview_order"].(string))
			d.Set("order", viewOrder)

			// Updating recursion mode
			if buf[0]["dnsview_recursion"].(string) == "yes" {
				d.Set("recursion", true)
			} else {
				d.Set("recursion", false)
			}

			d.Set("class", buf[0]["dnsview_class_name"].(string))

			// Updating local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["dnsview_class_parameters"].(string))
			computedClassParameters := map[string]string{}

			for ck := range currentClassParameters {
				if rv, rvExist := retrievedClassParameters[ck]; rvExist {
					computedClassParameters[ck] = rv[0]
				} else {
					computedClassParameters[ck] = ""
				}
			}

			d.Set("class_parameters", computedClassParameters)

			return nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				log.Printf("[DEBUG] SOLIDServer - Unable to find DNS server: %s (%s)\n", strings.ToLower(d.Get("name").(string)), errMsg)
			}
		} else {
			// Log the error
			log.Printf("[DEBUG] SOLIDServer - Unable to find DNS server (oid): %s\n", d.Id())
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return fmt.Errorf("SOLIDServer - Unable to find DNS server: %s\n", strings.ToLower(d.Get("name").(string)))
	}

	// Reporting a failure
	return err
}

func resourcednsviewImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("dns_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/dns_server_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("name", strings.ToLower(buf[0]["dnsview_name"].(string)))
			d.Set("dnsserver", buf[0]["dns_name"].(string))

			viewOrder, _ := strconv.Atoi(buf[0]["dnsview_order"].(string))
			d.Set("order", viewOrder)

			// Updating recursion mode
			if buf[0]["dnsview_recursion"].(string) == "yes" {
				d.Set("recursion", true)
			} else {
				d.Set("recursion", false)
			}

			d.Set("class", buf[0]["dnsview_class_name"].(string))

			// Updating local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["dnsview_class_parameters"].(string))
			computedClassParameters := map[string]string{}

			for ck := range currentClassParameters {
				if rv, rvExist := retrievedClassParameters[ck]; rvExist {
					computedClassParameters[ck] = rv[0]
				} else {
					computedClassParameters[ck] = ""
				}
			}

			d.Set("class_parameters", computedClassParameters)

			return []*schema.ResourceData{d}, nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				log.Printf("[DEBUG] SOLIDServer - Unable to import DNS view (oid): %s (%s)\n", d.Id(), errMsg)
			}
		} else {
			log.Printf("[DEBUG] SOLIDServer - Unable to find and import DNS view (oid): %s\n", d.Id())
		}

		// Reporting a failure
		return nil, fmt.Errorf("SOLIDServer - Unable to find and import DNS view (oid): %s\n", d.Id())
	}

	// Reporting a failure
	return nil, err
}