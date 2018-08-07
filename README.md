# stacker

Assemble cloudformation stacks by describing templates, parameters,
and interdependencies as configuration files.

## Usage

### Directory structure

An example stacker directory structure:

```
stacker
├── environments
│   ├── production
│   │   ├── api.yml
│   │   └── vpc.yml
│   ├── production.yml
│   ├── sandbox
│   │   ├── api.yml
│   │   └── vpc.yml
│   └── sandbox.yml
└── templates
    ├── API.yml
    ├── Database.yml
    ├── PublicSubnet.yml
    ├── PrivateSubnet.yml
    └── VPC.yml
```

#### environments/

The `environments/` directory contains files in the
[Environment file](#environment-file) format. This directory supports nesting
of configuration files, allowing for default stack attribute(s) and parameters
to be inherited from parent environment configurations. The above serves only as
an illustration of an example directory structure.

#### templates/

The `templates/` directory contains cloudformation template files in either
YAML or JSON format with the following supported extensions: `.yml`, `.yaml` and
`.json`.

### Environment file

```
defaults:
  region: us-east-1
  parameters:
    VpcCIDR: 10.101.0.0/16
    VpcId:
      Stack: NVA1-VPC.VpcId # Output from NVA1-VPC.VpcId
    InternetGateway:
      Stack: NVA1-VPC.InternetGateway
    PublicSubnets: &PublicSubnets
      - Stack: NVA1-PublicSubnetA.Subnet
      - Stack: NVA1-PublicSubnetB.Subnet

stacks:
  - name: NVA1-VPC
    template_name: VPC # templates/VPC.{yml, yaml, json}
    parameters:
      Name: VPCEast1
      EnableDnsSupport: true

  - name: NVA1-PublicSubnetA
    template_name: PublicSubnet
    parameters:
      CidrBlock: 10.101.10.0/24
      AvailabilityZone: us-east-1a

  - name: NVA1-PublicSubnetB
    template_name: PublicSubnet
    parameters:
      CidrBlock: 10.101.11.0/24
      AvailabilityZone: us-east-1b

  - name: API # The stack name implies the template # templates/API.{yml, yaml, json}
    capabilities: [CAPABILITY_IAM] # give permission to create IAM resources
    parameters:
      Subnets: *PublicSubnets
```

#### Stacks

Each stack configuration takes the following format:

```
- name: StackName
  region: us-east-1
  template_name: NameOfTemplate
  capabilities: [CAPABILITY_IAM]
  parameters:
    Name: BestStack # Literal, string parameter
    FileDataParam:
      File: /some/local/file # Parameter which passes the content of a local file
    StackOutputParam:
      Stack: VPCStack.VpcId # An output from the `VPCStack` in us-east-1 named `VpcId`
    CommaSeparatedListParam: # A comma-delimited list param
      - Foo
      - Bar
```

##### name

Name of the stack passed to cloudformation. This must be unique within a region.

##### region

The region the stack will be created within.

##### template_name

The template name correlates to a cloudformation template file located in
`templates/`. Stacker supports both JSON and YAML templates, and looks for a
template file with an extension of: `.yml`, `.yaml`, or `.yaml` within the templates
directory.

When this attribute is missing, Stacker will use the stack name as the template
name.

##### capabilities

Capabilities to provide the stack for creating resources.

Valid values include: `CAPABILITY_IAM`, `CAPABILITY_NAMED_IAM`


##### parameters

Parameters are supplied as a mapping of key to value. Lists of values can be
provided to comma-delimited list inputs.

In addition to literals, stacker provides a number of resolvers for dynamic parameters:

###### stack output

The stack output resolver will lookup the output value of an existing stack
within the same region and provide that as an input value. This is useful for
mapping resource dependencies across stacks.

For instance, a subnet will require the identifier of the Vpc
within which it resides:

```
- name: PrivateSubnetA
  region: us-east-1
  template: PrivateSubnet
  parameters:
    VpcId:
      Stack: VPC.VpcId
```

The above assumes a stack named `VPC` with an output of `VpcId` resides within
`us-east-1`.

###### file

The file resolver will pass the contents of a local file as a parameter value.


```
- name: StackA
  parameters:
    LocalFileParam:
      File: /path/to/param.txt
```

The above will pass the contents of `/path/to/param.txt` for the value of `LocalFileParam`.


#### Defaults

The defaults section describes defaults that are applied to all stacks
within an environment file. A top-level `region` may be supplied, as well as a
set of parameters.
