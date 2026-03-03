# Test Wide Mermaid Diagram

This file contains a wide Mermaid diagram to test rendering in the MarkdownPreview component.

## Architecture Diagram

```mermaid
graph TB
    subgraph "Client Layer"
        SunRun[SunRun Client<br/>🎯 Pilot Program]
        OtherClients[Other Clients<br/>Future Rollout]
    end
    
    subgraph "API Gateway"
        APIGW[ms-api-gateway<br/>Feature Flags & Routing]
        GraphQL[ms-graphql-gateway<br/>Federated GraphQL]
    end
    
    subgraph "New Route-Centric Services"
        Ingestion[ms-ingestion-engine<br/>Webhook Intake & Validation]
        Deliveries[ms-deliveries<br/>Delivery Processing & Shadow Mode]
        RoutePlanner[ms-route-planner<br/>Route Optimization & TSP]
        Automation[ms-automation-engine<br/>Custom Workflows]
    end
    
    subgraph "Enhanced Services"
        Monolith[ms-monolith<br/>Legacy Support]
        OrderQuote[ms-order-quote<br/>Route-Based Pricing]
        Locations[ms-locations<br/>Route Calculation]
    end
    
    subgraph "Driver Services"
        DriverEngagement[ms-driver-engagement<br/>Route Tracking]
        DriverTracker[ms-driver-order-tracker<br/>Multi-Stop Support]
        R2D2[ms-r2d2<br/>Route Notifications]
    end
    
    subgraph "New Data Model"
        OrderDB[("Order Database<br/>Order/Delivery/Route<br/>Shadow Tables")]
    end
    
    subgraph "Event-Driven Integration"
        Kafka[Kafka<br/>Event Streaming]
    end
    
    %% Data Flow
    SunRun --> APIGW
    OtherClients --> APIGW
    APIGW --> Ingestion
    Ingestion --> Deliveries
    APIGW --> OrderQuote
    APIGW --> Locations
    
    Deliveries --> RoutePlanner
    Deliveries --> Automation
    RoutePlanner --> Automation
    
    Ingestion --> OrderDB
    Deliveries --> OrderDB
    RoutePlanner --> OrderDB
    Automation --> OrderDB
    
    Ingestion --> Kafka
    Deliveries --> Kafka
    RoutePlanner --> Kafka
    Automation --> Kafka
    
    Kafka --> DriverEngagement
    Kafka --> DriverTracker
    Kafka --> R2D2
    
    DriverEngagement --> OrderDB
    DriverTracker --> OrderDB
```

## Expected Behavior

The diagram above should:
1. Be fully visible in the preview pane
2. Show horizontal scrollbars if wider than the container
3. Be clickable to expand in a modal

