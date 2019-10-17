/*
Copyright 2019 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"encoding/base64"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

/* Kubernetes and Kabanero yaml constants*/
const (
	V1 = "v1" 
	V1ALPHA1 = "v1alpha1"
	KABANEROIO = "kabanero.io"
	KABANERO = "kabanero"
	KABANEROS = "kabaneros"
	DATA = "data"
    URL = "url"
	USERNAME = "username"
	TOKEN = "token"
	SECRETS = "secrets"
	SPEC = "spec"
	COLLECTIONS = "collections"
	REPOSITORIES = "repositories"
	ACTIVATEDEFAULTCOLLECTIONS = "activateDefaultCollections" 
)

/*
 Find the user/token for a Github APi KEy
 The format of the secret:
    apiVersion: v1
    kind: Secret
    metadata:
      name: <name of secret>
    type: Opaque
	data:
      url: https://github.com/org  or https://github.com/org/repo
      username: base64 encoded user
	  token: base64 encoded token

 If the url in the secret is a prefix of repoURL, and username and token are defined, then return the user and token.
 Return user, token, error. 
 TODO: Change to controller pattern and cache the secrets.
 */
func getURLAPIToken(dynInterf dynamic.Interface, namespace string, repoURL string) (string, string, error){
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  V1,
		Resource: SECRETS,
	}
	var intfNoNS = dynInterf.Resource(gvr)
	var intf dynamic.ResourceInterface
	intf = intfNoNS.Namespace(namespace)

	// fetch the current resource
	var unstructuredList *unstructured.UnstructuredList
	var err error
	unstructuredList, err = intf.List(metav1.ListOptions{})
	if err != nil {
		return "", "", err
	}

	for _, unstructuredObj := range unstructuredList.Items {
	    var objMap = unstructuredObj.Object
		dataMapObj, ok := objMap[DATA]
		if !ok {
			continue
		}

    	dataMap, ok := dataMapObj.(map[string]interface{})
    	if !ok {
    		continue
		}

		urlObj, ok := dataMap[URL]
		if !ok {
			continue
		}
		url, ok := urlObj.(string)
		if !ok {
			continue
		}

		if !strings.HasPrefix(repoURL, url) {
            /* not for this uRL */
            continue
		}

		usernameObj, ok := dataMap[USERNAME]
		if !ok {
			continue
		}
		username, ok := usernameObj.(string)
		if !ok {
			continue
		}

		tokenObj, ok := dataMap[TOKEN]
		if !ok {
			continue
		}
		token, ok := tokenObj.(string)
		if !ok {
			continue
		}

		decodedUserName, err := base64.StdEncoding.DecodeString(username)
		if err != nil {
			return "", "", err
		}

		decodedToken, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return "", "", err
		}
		return string(decodedUserName), string(decodedToken), nil
	}
	return "", "", fmt.Errorf("Unable to find API token for url: %s", repoURL)
}

/* Get the URL to kabanero-index.yaml
*/
func getKabaneroIndexURL(dynInterf dynamic.Interface, namespace string) (string, error){
        if klog.V(5) {
            klog.Infof("Entering getKabaneroIndexURL")
            defer klog.Infof("Leaving getKabaneroIndexURL")
        }

	gvr := schema.GroupVersionResource{
		Group:    KABANEROIO,
		Version:  V1ALPHA1,
		Resource: KABANEROS,
	}
	var intfNoNS = dynInterf.Resource(gvr)
	var intf dynamic.ResourceInterface
	intf = intfNoNS.Namespace(namespace)

	// fetch the current resource
	var unstructuredList *unstructured.UnstructuredList
	var err error
	unstructuredList, err = intf.List(metav1.ListOptions{})
	if err != nil {
                klog.Errorf("Unable to list resource of kind kabanero in the namespace %s",namespace) 
		return "", err
	}

	for _, unstructuredObj := range unstructuredList.Items {
            if klog.V(5) {
                klog.Infof("Processing kabanero CRD instance: %v", unstructuredObj);
            }
	    var objMap = unstructuredObj.Object
		specMapObj, ok := objMap[SPEC]
		if !ok {
                    if klog.V(5) {
                        klog.Infof("    kabanero CRD instance: has no spec section. Skipping");
                    }
                    continue
		}

    	specMap, ok := specMapObj.(map[string]interface{})
    	if !ok {
                    if klog.V(5) {
                        klog.Infof("    kabanero CRD instance: spec section is type %T. Skipping", specMapObj);
                    }
    		continue
		}

		collectionsMapObj, ok := specMap[COLLECTIONS]
		if !ok {
                    if klog.V(5) {
                        klog.Infof("    kabanero CRD instance: spec section has no collections section. Skipping");
                    }
                    continue
		}
		collectionMap, ok := collectionsMapObj.(map[string]interface{})
		if !ok {
                    if klog.V(5) {
                        klog.Infof("    kabanero CRD instance: collections type is %T. Skipping", collectionsMapObj);
                    }
                    continue
		}

		repositoriesInterface, ok := collectionMap[REPOSITORIES]
		if !ok {
                    if klog.V(5) {
                        klog.Infof("    kabanero CRD instance: collections section has no repositories section. Skipping");
                    }
			continue
		}
		repositoriesArray, ok :=repositoriesInterface.([]interface{})
		if !ok {
                    if klog.V(5) {
                        klog.Infof("    kabanero CRD instance: repositories  type is %T. Skipping", repositoriesInterface);
                    }
                    continue
		}
		for index, elementObj := range repositoriesArray {
		  elementMap, ok := elementObj.(map[string] interface{})
		  if !ok {
                      if klog.V(5) {
                          klog.Infof("    kabanero CRD instance repositories index %d, types is %T. Skipping", index, elementObj);
                      }
                      continue
		  }
		  activeDefaultCollectionsObj, ok := elementMap[ACTIVATEDEFAULTCOLLECTIONS]
		  if !ok {
                        if klog.V(5) {
                            klog.Infof("    kabanero CRD instance: index %d, activeDefaultCollection not set. Skipping", index)
                        }
			  continue
		  }
		  active, ok := activeDefaultCollectionsObj.(bool)
		  if !ok {
                      if klog.V(5) {
                          klog.Infof("    kabanero CRD instance index %d, activeDefaultCollection, types is %T. Skipping", activeDefaultCollectionsObj);
                      }
			  continue
		  }
		  if active {
			  urlObj, ok := elementMap[URL]
			  if !ok {
                              if klog.V(5) {
                                  klog.Infof("    kabanero CRD instance: index %d, url set. Skipping", index)
                              }
                              continue
			  }
			  url, ok := urlObj.(string)
			  if !ok {
                              if klog.V(5) {
                                  klog.Infof("    kabanero CRD instance index %d, url type is %T. Skipping", url);
                              }
                              continue
			  }
			  return url, nil
		  }
		}
	}
	return "", fmt.Errorf("Unable to find collection url in kabanero custom resource for namespace %s", namespace)
}