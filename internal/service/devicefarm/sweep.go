//go:build sweep
// +build sweep

package devicefarm

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/devicefarm"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep"
)

func init() {
	resource.AddTestSweepers("aws_devicefarm_project", &resource.Sweeper{
		Name: "aws_devicefarm_project",
		F:    sweepProjects,
	})
}

func sweepProjects(region string) error {
	client, err := sweep.SharedRegionalSweepClient(region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.(*conns.AWSClient).DeviceFarmConn
	sweepResources := make([]*sweep.SweepResource, 0)
	var errs *multierror.Error

	input := &devicefarm.ListProjectsInput{}

	err = conn.ListProjectsPages(input, func(page *devicefarm.ListProjectsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, project := range page.Projects {
			r := ResourceProject()
			d := r.Data(nil)

			id := aws.StringValue(project.Arn)
			d.SetId(id)

			if err != nil {
				err := fmt.Errorf("error reading DeviceFarm Project (%s): %w", id, err)
				log.Printf("[ERROR] %s", err)
				errs = multierror.Append(errs, err)
				continue
			}

			sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error listing DeviceFarm Project for %s: %w", region, err))
	}

	if err := sweep.SweepOrchestrator(sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping DeviceFarm Project for %s: %w", region, err))
	}

	if sweep.SkipSweepError(err) {
		log.Printf("[WARN] Skipping DeviceFarm Project sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}
