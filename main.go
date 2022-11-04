package main

import (
	"context"
	"log"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// Retrieve cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to retrieve in-cluster configuration: %v\n", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create new clientset: %v\n", err)
	}

	// Loop through and get deployments
	for {
		currentDay := time.Now().Weekday().String()
		deployments, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Failed to retrieve deployments: %v\n", err)
		}
		log.Printf("Processing %v deployments in the cluster\n", len(deployments.Items))

		for _, deployment := range deployments.Items {
			log.Printf("Working on %v in namespace %v", deployment.Name, deployment.Namespace)
			labels := deployment.Labels

			// Retrieve Deployment Schedule
			// Valid Schedules: Mon-Sat-Sun
			if schedule, ok := labels["schedule"]; ok {
				activeDays := strings.Split(schedule, "-")
				scaleToZero := true

				// Check if we should scale down the deployment
				for _, day := range activeDays {
					scaleToZero = !strings.HasPrefix(day, currentDay)
				}

				if scaleToZero {
					// TODO: Save current scale somewhere are retrieve it when we need to scale up again
					// Note: HPAs are disabled when scaled down to 0 until scaled back to original value
					deploymentInterface := clientset.AppsV1().Deployments(deployment.Name)
					currentScale, err := deploymentInterface.GetScale(context.TODO(), deployment.Name, metav1.GetOptions{})
					if err != nil {
						log.Fatalf("Failed to get current scale of the %v deployment:\n%v\n", deployment.Name, err)
					}
					newScale := *currentScale
					newScale.Spec.Replicas = 0

					log.Printf("Scaling down the %v deployment", deployment.Name)
					updatedScale, err := deploymentInterface.UpdateScale(context.TODO(), deployment.Name, &newScale, metav1.UpdateOptions{})
					if err != nil {
						log.Fatalf("Failed to scale down the %v deployment:\n%v\n", deployment.Name, err)
					}
					log.Printf("Scaled down the %v deployment:\n%v\n", deployment.Name, updatedScale)
				}
			}
		}

		// Sleep for 30 seconds per scan... Probably want less
		time.Sleep(time.Second * 30)
	}
}
