# dfsr
Windows Distributed File System Replication monitoring for the Go programming
language

[![GoDoc](https://godoc.org/gopkg.in/dfsr.v0?status.svg)](https://godoc.org/gopkg.in/dfsr.v0)

This is a repository of Go packages, command line tools and services that allow
for manual and automated monitoring of DFSR configuration and backlog counts.

Please note that these packages and tools are still `v0`, and as such the
implementations and APIs may change in the future.

## Windows DFSR Monitor Service

A windows service is included in the `svc/dfsrmonitor` package that is
capable of monitoring replication group backlogs domain-wide and reporting the
values to [StatHat](https://www.stathat.com/).

While StatHat is the only supported backlog consumer at this time, others may
be added in the future. See the `monitor/consumer/stathatconsumer` package
for the source of the StatHat implementation.

The service is designed to query DFSR configuration and backlogs more
efficiently than traditional `powershell` scripts or the `dfsrdiag` tool.
Queries are executed in parallel, and configuration data and version vectors are
cached appropriately to avoid unnecessary querying. Queries to each individual
server are queued and serialized to avoid overburdening the members during
intensive DFSR initialization and recovery tasks.

### Windows Service Installation

```
go get -u gopkg.in/dfsr.v0/svc/dfsrmonitor
go build gopkg.in/dfsr.v0/svc/dfsrmonitor
dfsrmonitor install <flags>
```

Run `dfsrmonitor install -h` to see a list of possible command line flags.

Run `dfsrmonitor debug` to test the monitor as a command line program.

**TODO**: Discuss Windows Service identity options

**TODO**: Describe StatHat name formats

**TODO**: Provide example installation commands

**KNOWN BUG**: The installed Windows service does not currently start with the
               parameters specified during installation.

### Windows Service Access Control Configuration

Microsoft's guidance regarding privileges needed to calculate backlogs is to
run everything as a `Domain Admin` account. While that may be suitable for
manual operation by system administrators, in the context of an automated
service the package authors consider this guidance a sad divergence from the
the principle of least privilege.

The instructions below demonstrate how to create a domain account with the
minimal set of privileges necessary to monitor DFSR backlog information for a
domain. They have been tested on `Windows Server 2012 R2` and
`Windows Server 2016`.

#### Create domain account and/or security groups

Create a new *domain account* under which the DFSR Monitor will operate.

You may additionally create a domain local security group and add the
*domain account* as a member of the new group. In most cases this is preferable
because it allows for additional or distinct memberships without editing the
complex access controls listed below. If you do this you should use the security
group instead of the domain account in the directions below, whenever it
mentions a *domain account*.

#### Add the domain account to the `Distributed COM Users` local security group

Domain accounts must be included in the well-known `Distributed COM Users`
group in order for them to make DCOM calls to a remote server.

Perform the following steps on each DFSR member:

1. Open a command prompt as an administrator
2. Run `compmgmt.msc`
3. Expand `System Tools`, `Local Users and Groups` and select `Groups`
4. Open the `Distributed COM Users` local security group
5. Add the *domain account* as a member of the group

These steps can be automated via Group Policy.

#### Enable security changes for the `DFSRHelper.ServerHealthReport Class`

The *domain account* must be allowed to talk to the DFSR Helper protocol server,
but modification of its access control list is normally limited to the Windows
`TrustedInstaller` principal. In order to modify its launch and activation
rights, the `Administrators` group must first be allowed to make changes to it.
This can be accomplished by taking ownership of a Windows registry entry and
granting `Full Control` to the local `Administrators` group.

Perform the following steps on each DFSR member:

1. Run `regedit` as an administrator
2. Navigate to the `HKEY_LOCAL_MACHINE\SOFTWARE\Classes\AppID\{36C95A5F-0A17-47c7-9983-F6BFD009A867}` node
3. Right-click the `{36C95A5F-0A17-47c7-9983-F6BFD009A867}` node and select `Permissions...`
4. Click on the `Advanced` button in the bottom right
5. If the `Owner` is currently `TrustedInstaller` it should be changed to `Administrators` to allow editing by members of the group:
   1. Click the `Change` button beside the owner field to open up the `Select User or Group` dialog
   2. Click the `Locations...` button, choose the local machine, then click `OK`
   3. Enter `Administrators` in the object name field and then click `OK`
   4. Check the `Replace owner on subcontainers and objects` box
   5. Click `Apply` to update the owner
6. Edit the `Administrators` entry, grant `Full Control` by checking the box, then click `OK`
7. Click `Apply` or `OK` for the updated permissions to take effect

These steps can be automated via Group Policy. The simplest approach is to
`allow inheritable permissions from the parent to propagate to the key and to
all child objects` and to `replace existing permissions on all subkeys with
inheritable permissions`. Doing so will cause the key to inherit ACLs that
grant the `Administrators` group `Full Control`.

#### Grant the *domain account* access to the `DFSRHelper.ServerHealthReport Class`

Perform the following steps on each DFSR member:

1. Run `comexp.msc`
2. Expand the `Component Services`, `Computers`, `My Computer` nodes and then select the `DCOM Config` node
3. Change the current view mode to `Detail`
4. Locate the `DFSRHelper.ServerHealthReport Class` with the `{36C95A5F-0A17-47c7-9983-F6BFD009A867}` Application ID, right-click it and select `Properties`
5. Navigate to the `Security` tab
6. Change the `Launch and Activation Permissions` to `Customize` then click `Edit` to open the `Launch and Activation Permission` window
7. Add the desired *domain account* and allow it the `Local Launch`, `Remote Launch`, `Local Activation` and `Remote Activation` permissions
8. Click `OK` for the updated permissions to take effect

These steps can be automated via Group Policy. The simplest approach is to
perform the modification once manually, then export the affected
`LaunchPermission` value from the
`HKEY_LOCAL_MACHINE\SOFTWARE\Classes\AppID\{36C95A5F-0A17-47c7-9983-F6BFD009A867}`
registry key. The `LaunchPermission` `REG_BINARY` value can be distributed via
group policy preferences.

#### Grant the *domain account* access to WMI performance counters

The current implementation of the DFSR Helper protocol server relies upon the
WMI framework internally, and interacts with the framework with the security
context of the calling user. This means that the domain must be granted
remote execution privileges for the `MicrosoftDfs` WMI namespace.

Perform the following steps on each DFSR member:

1. Open a command prompt as an administrator
2. Run `wmimgmt.msc`
3. Right-click on `WMI Control (Local)` and select `Properties`
4. Navigate to the `Security` tab
5. Expand the `Root` node
6. Click on `MicrosoftDfs` and then click the `Security` button in the bottom right
7. Add the desired *domain account* and allow it the `Execute Methods`, `Enable Account` and `Remote Enable` permissions
8. Click `Apply` or `OK` for the updated permissions to take effect

These steps cannot be automated via Group Policy.

#### Grant the *domain account* delegated access to DFS replication groups

Perform the following steps from any workstation with the
[Remote Server Administration Tools](https://www.microsoft.com/en-us/download/details.aspx?id=45520) installed:

1. Open a command prompt as a domain administrator
2. Run `dfsmgmt.msc`
3. If necessary, right-click on the `Replication` node and `add replication groups for display`, then select some or all replications groups and click `OK`
4. Select the `Replication` node
5. In the side menu on the far right, click the `Delegate Management Permissions...` link
6. Click `Add`
7. Enter the name of the *domain account* and then click `OK` for the updated permissions to take effect
