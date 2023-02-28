# ACSCS Network (VPC) Diagram

- Diagram represents one environment (e.g. Stage, Prod)
- OSD Clusters have their own VPC provisioned at creation time
  - We leave their CIDR range as the default (10.0.0.0/16)
  - This means that they *can not* be networked together

```mermaid
graph
    direction TB

    subgraph dps[Data Plane OSD Clusters]
        subgraph vpcdp01[VPC CIDR: 10.0.0.0/16]
            subgraph dp01[acs-prod-dp-01]
                central-01-1
                central-01-2
            end
        end

        subgraph vpcdp02[VPC CIDR: 10.0.0.0/16]
            subgraph dp02[acs-prod-dp-02 CIDR: 10.0.0.0/16]
                central-02-1
                central-02-2
            end
        end
    end

    subgraph vpcrds[Database VPC CIDR: 10.1.0.0/16]
        subgraph rds[RDS Instances]
            rds-01-1["RDS 01-1 ⛁"]
            rds-01-2["RDS 01-2 ⛁"]
            rds-02-1["RDS 02-1 ⛁"]
            rds-02-2["RDS 02-2 ⛁"]
        end
    end

    central-01-1 --> rds-01-1
    central-01-2 --> rds-01-2
    central-02-1 --> rds-02-1
    central-02-2 --> rds-02-2    
```
