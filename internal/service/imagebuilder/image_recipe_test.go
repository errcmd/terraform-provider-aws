package imagebuilder_test

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/imagebuilder"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/go-multierror"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfimagebuilder "github.com/hashicorp/terraform-provider-aws/internal/service/imagebuilder"
)

func init() {
	resource.AddTestSweepers("aws_imagebuilder_image_recipe", &resource.Sweeper{
		Name: "aws_imagebuilder_image_recipe",
		F:    testSweepImageBuilderImageRecipes,
	})
}

func testSweepImageBuilderImageRecipes(region string) error {
	client, err := acctest.SharedRegionalSweeperClient(region)
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	conn := client.(*conns.AWSClient).ImageBuilderConn

	var sweeperErrs *multierror.Error

	input := &imagebuilder.ListImageRecipesInput{
		Owner: aws.String(imagebuilder.OwnershipSelf),
	}

	err = conn.ListImageRecipesPages(input, func(page *imagebuilder.ListImageRecipesOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, imageRecipeSummary := range page.ImageRecipeSummaryList {
			if imageRecipeSummary == nil {
				continue
			}

			arn := aws.StringValue(imageRecipeSummary.Arn)

			r := tfimagebuilder.ResourceImageRecipe()
			d := r.Data(nil)
			d.SetId(arn)

			err := r.Delete(d, client)

			if err != nil {
				sweeperErr := fmt.Errorf("error deleting Image Builder Image Recipe (%s): %w", arn, err)
				log.Printf("[ERROR] %s", sweeperErr)
				sweeperErrs = multierror.Append(sweeperErrs, sweeperErr)
				continue
			}
		}

		return !lastPage
	})

	if acctest.SkipSweepError(err) {
		log.Printf("[WARN] Skipping Image Builder Image Recipe sweep for %s: %s", region, err)
		return sweeperErrs.ErrorOrNil() // In case we have completed some pages, but had errors
	}

	if err != nil {
		sweeperErrs = multierror.Append(sweeperErrs, fmt.Errorf("error listing Image Builder Image Recipes: %w", err))
	}

	return sweeperErrs.ErrorOrNil()
}

func TestAccImageBuilderImageRecipe_basic(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeNameConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "imagebuilder", regexp.MustCompile(fmt.Sprintf("image-recipe/%s/1.0.0", rName))),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "component.#", "1"),
					acctest.CheckResourceAttrRFC3339(resourceName, "date_created"),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					acctest.CheckResourceAttrAccountID(resourceName, "owner"),
					acctest.CheckResourceAttrRegionalARNAccountID(resourceName, "parent_image", "imagebuilder", "aws", "image/amazon-linux-2-x86/x.x.x"),
					resource.TestCheckResourceAttr(resourceName, "platform", imagebuilder.PlatformLinux),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "version", "1.0.0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_disappears(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeNameConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					acctest.CheckResourceDisappears(acctest.Provider, tfimagebuilder.ResourceImageRecipe(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMapping_deviceName(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingDeviceNameConfig(rName, "/dev/xvdb"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"device_name": "/dev/xvdb",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMappingEBS_deleteOnTermination(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingEBSDeleteOnTerminationConfig(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"ebs.0.delete_on_termination": "true",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMappingEBS_encrypted(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingEBSEncryptedConfig(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"ebs.0.encrypted": "true",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMappingEBS_iops(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingEBSIopsConfig(rName, 100),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"ebs.0.iops": "100",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMappingEBS_kmsKeyID(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	kmsKeyResourceName := "aws_kms_key.test"
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingEBSKMSKeyIDConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "block_device_mapping.*.ebs.0.kms_key_id", kmsKeyResourceName, "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMappingEBS_snapshotID(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	ebsSnapshotResourceName := "aws_ebs_snapshot.test"
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingEBSSnapshotIDConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "block_device_mapping.*.ebs.0.snapshot_id", ebsSnapshotResourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMappingEBS_volumeSize(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingEBSVolumeSizeConfig(rName, 20),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"ebs.0.volume_size": "20",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMappingEBS_volumeTypeGP2(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingEBSVolumeTypeConfig(rName, imagebuilder.EbsVolumeTypeGp2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"ebs.0.volume_type": imagebuilder.EbsVolumeTypeGp2,
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMappingEBS_volumeTypeGP3(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingEBSVolumeTypeConfig(rName, tfimagebuilder.EBSVolumeTypeGP3),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"ebs.0.volume_type": tfimagebuilder.EBSVolumeTypeGP3,
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMapping_noDevice(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingNoDeviceConfig(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"no_device": "true",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_BlockDeviceMapping_virtualName(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeBlockDeviceMappingVirtualNameConfig(rName, "ephemeral0"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "block_device_mapping.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "block_device_mapping.*", map[string]string{
						"virtual_name": "ephemeral0",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_component(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeComponentConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "component.#", "3"),
					resource.TestCheckResourceAttrPair(resourceName, "component.0.component_arn", "data.aws_imagebuilder_component.aws-cli-version-2-linux", "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "component.1.component_arn", "data.aws_imagebuilder_component.update-linux", "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "component.2.component_arn", "aws_imagebuilder_component.test", "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_description(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeDescriptionConfig(rName, "description1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "description1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_tags(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeTags1Config(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccImageRecipeTags2Config(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccImageRecipeTags1Config(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccImageBuilderImageRecipe_workingDirectory(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_imagebuilder_image_recipe.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, imagebuilder.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckImageRecipeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccImageRecipeWorkingDirectoryConfig(rName, "/tmp"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageRecipeExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "working_directory", "/tmp"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckImageRecipeDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).ImageBuilderConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_imagebuilder_image_recipe" {
			continue
		}

		input := &imagebuilder.GetImageRecipeInput{
			ImageRecipeArn: aws.String(rs.Primary.ID),
		}

		output, err := conn.GetImageRecipe(input)

		if tfawserr.ErrCodeEquals(err, imagebuilder.ErrCodeResourceNotFoundException) {
			continue
		}

		if err != nil {
			return fmt.Errorf("error getting Image Builder Image Recipe (%s): %w", rs.Primary.ID, err)
		}

		if output != nil {
			return fmt.Errorf("Image Builder Image Recipe (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckImageRecipeExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).ImageBuilderConn

		input := &imagebuilder.GetImageRecipeInput{
			ImageRecipeArn: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetImageRecipe(input)

		if err != nil {
			return fmt.Errorf("error getting Image Builder Image Recipe (%s): %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccImageRecipeBaseConfig(rName string) string {
	return fmt.Sprintf(`
data "aws_region" "current" {}

data "aws_partition" "current" {}

resource "aws_imagebuilder_component" "test" {
  data = yamlencode({
    phases = [{
      name = "build"
      steps = [{
        action = "ExecuteBash"
        inputs = {
          commands = ["echo 'hello world'"]
        }
        name      = "example"
        onFailure = "Continue"
      }]
    }]
    schemaVersion = 1.0
  })
  name     = %[1]q
  platform = "Linux"
  version  = "1.0.0"
}
`, rName)
}

func testAccImageRecipeBlockDeviceMappingDeviceNameConfig(rName string, deviceName string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    device_name = %[2]q
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, deviceName))
}

func testAccImageRecipeBlockDeviceMappingEBSDeleteOnTerminationConfig(rName string, deleteOnTermination bool) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    ebs {
      delete_on_termination = %[2]t
    }
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, deleteOnTermination))
}

func testAccImageRecipeBlockDeviceMappingEBSEncryptedConfig(rName string, encrypted bool) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    ebs {
      encrypted = %[2]t
    }
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, encrypted))
}

func testAccImageRecipeBlockDeviceMappingEBSIopsConfig(rName string, iops int) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    ebs {
      iops = %[2]d
    }
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, iops))
}

func testAccImageRecipeBlockDeviceMappingEBSKMSKeyIDConfig(rName string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_kms_key" "test" {
  deletion_window_in_days = 7
}

resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    ebs {
      kms_key_id = aws_kms_key.test.arn
    }
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName))
}

func testAccImageRecipeBlockDeviceMappingEBSSnapshotIDConfig(rName string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		acctest.ConfigAvailableAZsNoOptIn(),
		fmt.Sprintf(`
resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  size              = 1
}

resource "aws_ebs_snapshot" "test" {
  volume_id = aws_ebs_volume.test.id
}

resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    ebs {
      snapshot_id = aws_ebs_snapshot.test.id
    }
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName))
}

func testAccImageRecipeBlockDeviceMappingEBSVolumeSizeConfig(rName string, volumeSize int) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    ebs {
      volume_size = %[2]d
    }
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, volumeSize))
}

func testAccImageRecipeBlockDeviceMappingEBSVolumeTypeConfig(rName string, volumeType string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    ebs {
      volume_type = %[2]q
    }
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, volumeType))
}

func testAccImageRecipeBlockDeviceMappingNoDeviceConfig(rName string, noDevice bool) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    no_device = %[2]t
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, noDevice))
}

func testAccImageRecipeBlockDeviceMappingVirtualNameConfig(rName string, virtualName string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  block_device_mapping {
    virtual_name = %[2]q
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, virtualName))
}

func testAccImageRecipeComponentConfig(rName string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
data "aws_imagebuilder_component" "aws-cli-version-2-linux" {
  arn = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:component/aws-cli-version-2-linux/1.0.0"
}

data "aws_imagebuilder_component" "update-linux" {
  arn = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:component/update-linux/1.0.0"
}

resource "aws_imagebuilder_image_recipe" "test" {
  component {
    component_arn = data.aws_imagebuilder_component.aws-cli-version-2-linux.arn
  }

  component {
    component_arn = data.aws_imagebuilder_component.update-linux.arn
  }

  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName))
}

func testAccImageRecipeDescriptionConfig(rName string, description string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  description  = %[2]q
  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName, description))
}

func testAccImageRecipeNameConfig(rName string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"
}
`, rName))
}

func testAccImageRecipeTags1Config(rName string, tagKey1 string, tagValue1 string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1))
}

func testAccImageRecipeTags2Config(rName string, tagKey1 string, tagValue1 string, tagKey2 string, tagValue2 string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name         = %[1]q
  parent_image = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version      = "1.0.0"

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2))
}

func testAccImageRecipeWorkingDirectoryConfig(rName string, workingDirectory string) string {
	return acctest.ConfigCompose(
		testAccImageRecipeBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_imagebuilder_image_recipe" "test" {
  component {
    component_arn = aws_imagebuilder_component.test.arn
  }

  name              = %[1]q
  parent_image      = "arn:${data.aws_partition.current.partition}:imagebuilder:${data.aws_region.current.name}:aws:image/amazon-linux-2-x86/x.x.x"
  version           = "1.0.0"
  working_directory = %[2]q
}
`, rName, workingDirectory))
}
